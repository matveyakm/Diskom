package restController

// Файловые приложения сущностей: отдельная таблица `files` в БД каждого
// микросервиса (решение: файлы живут там же, где их хозяин).
//
// Связь: files.id_document → id строки основной таблицы (1:N, NULL =
// сирота: загружен, но ни к чему не привязан). Один файл — одно поле
// формы; несколько файлов документа — несколько строк.
//
// Поток (договорённость):
//  1. POST /api/files {file_name, data(base64)} → orphan-строка {id...}.
//     Отдельный запрос (файл НЕ едет внутри JSON документа).
//  2. Сохранение документа: пайплайн PATCH /api/files?id=<id_file>
//     {id_document: <id>} — привязка. Привязать можно только сироту
//     (чужой файл увести нельзя); повторная привязка к тому же документу
//     — тихий no-op (идемпотентный re-save). ?replace=1 — сначала отцепить
//     остальные файлы документа (смена одиночного логотипа).
//  3. GET /api/files?id_document= — МЕТАДАННЫЕ без байтов (грид и формы).
//     GET /api/files?content= — байты с Content-Type/Disposition.
//  4. DELETE /api/files?id= — удаление строки. Привязанные файлы чистятся
//     шагом on_delete сущности (DELETE /api/files?id_document= —
//     фильтрованный multi-DELETE движок уже умеет).
//  5. Сироты: created_at + TTL (env FILE_ORPHAN_TTL_HOURS, дефолт 168).
//     Чистит фоновый сборщик в App (тихий DELETE по таймеру, дефолт 6ч).
//     Сирота = id_document IS NULL ИЛИ номер документа исчез из основной
//     таблицы (проверка JOIN при чистке, руками файл не теряется).
//
// Байты в списках не отдаются никогда (правило); кап размера —
// env FILE_MAX_MB (дефолт 25); magic allowlist png/jpg/pdf (как в печати).

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Дефолты файлового механизма (переопределяются env).
const (
	defaultFileMaxMB            = 25
	defaultFileOrphanTTL        = 168 * time.Hour // 7 суток
	defaultFileGCInterval       = 6 * time.Hour
	fileGCStartupDelay          = time.Minute
	maxFileNameLen              = 255
	fileBodySlackBytes    int64 = 2 << 20 // запас поверх base64-раздувания
	// attachMaxBodyBytes — кап тела привязки (три скалярных поля, файлам
	// отдельный endpoint; как updateMaxBodyBytes в restController).
	attachMaxBodyBytes int64 = 1 << 16
)

// FileObject — строка таблицы files (одна таблица на БД каждого сервиса).
// Data в JSON наружу не отдаётся никогда (json:"-"; метаданные —
// отдельным запросом без колонки).
type FileObject struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FileName   string    `gorm:"column:file_name" json:"file_name"`
	Mime       string    `gorm:"column:mime" json:"mime"`
	Size       int64     `gorm:"column:size" json:"size"`
	Data       []byte    `gorm:"column:data" json:"-"`
	IdDocument *int64    `gorm:"column:id_document" json:"id_document"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName — фиксированное имя (одинаковое во всех БД).
func (FileObject) TableName() string { return "files" }

// fileMeta — метаданные без байтов (единственный вид списка).
type fileMeta struct {
	Id         int64     `json:"id"`
	FileName   string    `json:"file_name"`
	Mime       string    `json:"mime"`
	Size       int64     `json:"size"`
	IdDocument *int64    `json:"id_document"`
	CreatedAt  time.Time `json:"created_at"`
}

// fileUpload — тело загрузки (отдельный запрос, не часть документа):
// либо {file_name, data: base64}, либо {copy_from: id, id_document?}.
type fileUpload struct {
	FileName   string `json:"file_name"`
	Data       string `json:"data"`
	CopyFrom   any    `json:"copy_from"`
	IdDocument *int64 `json:"id_document"`
}

// FileHandler обслуживает /api/files одного микросервиса.
// parentTable — имя основной таблицы сервиса (для проверки FK и GC).
type FileHandler struct {
	db          *gorm.DB
	parentTable string
}

// NewFileHandler собирает обработчик файлов сервиса.
func NewFileHandler(db *gorm.DB, parentTable string) *FileHandler {
	return &FileHandler{db: db, parentTable: parentTable}
}

// EnsureFileTable создаёт таблицу files, если её нет (все БД, включая
// существующие volume — initdb-скрипты для них уже не выполнятся).
// Вызывается один раз при старте сервиса.
func EnsureFileTable(db *gorm.DB) error {
	return db.AutoMigrate(&FileObject{})
}

// fileMaxBytes — кап decoded-размера (env FILE_MAX_MB, дефолт 25).
func fileMaxBytes() int64 {
	if v := strings.TrimSpace(os.Getenv("FILE_MAX_MB")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 1 && n <= 1024 {
			return n << 20
		}
	}
	return defaultFileMaxMB << 20
}

// fileOrphanTTL — возраст сироты для чистки (env FILE_ORPHAN_TTL_HOURS).
func fileOrphanTTL() time.Duration {
	if v := strings.TrimSpace(os.Getenv("FILE_ORPHAN_TTL_HOURS")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 1 && n <= 24*365 {
			return time.Duration(n) * time.Hour
		}
	}
	return defaultFileOrphanTTL
}

// fileGCInterval — период сборщика (env FILE_GC_INTERVAL_HOURS).
func fileGCInterval() time.Duration {
	if v := strings.TrimSpace(os.Getenv("FILE_GC_INTERVAL_HOURS")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 1 && n <= 24*30 {
			return time.Duration(n) * time.Hour
		}
	}
	return defaultFileGCInterval
}

// fileMimeByMagic определяет mime по magic (allowlist; как в печати:
// доверяем байтам, а не расширению/заявленному типу).
func fileMimeByMagic(raw []byte) (string, bool) {
	if len(raw) < 4 {
		return "", false
	}
	if raw[0] == 0x89 && raw[1] == 'P' && raw[2] == 'N' && raw[3] == 'G' {
		return "image/png", true
	}
	if raw[0] == 0xFF && raw[1] == 0xD8 {
		return "image/jpeg", true
	}
	if len(raw) >= 5 && string(raw[:5]) == "%PDF-" {
		return "application/pdf", true
	}
	return "", false
}

// sanitizeFileName чистит имя (basename, без путей; длину режет вызывающий
// валидацией; пустое после чистки — ошибка).
func sanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "\\", "/")
	base := strings.TrimSpace(filepath.Base(name))
	// Точки-ссылки на каталог — не имена ("." уже ловится вызывающим,
	// ".." отдавал бы Content-Disposition с именем каталога).
	if base == "." || base == ".." || base == "/" {
		return ""
	}
	return base
}

// HandleRequest роутит /api/files (мультиплекс по query — как ?print=
// на основном роуте: новых путей не заводим).
func (h *FileHandler) HandleRequest(w http.ResponseWriter, req *http.Request) {
	// writeJSONResponse/writeError — общие хелперы CRUD-репозитория.
	r := &CRUDRepository[FileObject]{db: h.db, tableName: FileObject{}.TableName()}
	q := req.URL.Query()
	switch req.Method {
	case http.MethodPost:
		h.upload(w, req, r)
	case http.MethodGet:
		if id := q.Get("content"); id != "" {
			h.content(w, req, r, id)
			return
		}
		h.list(w, req, r)
	case http.MethodPut, http.MethodPatch:
		h.attach(w, req, r)
	case http.MethodDelete:
		h.remove(w, req, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// upload принимает файл отдельным запросом (сирота, без id_document).
// Вариант {"copy_from": <id>, "id_document"?: <doc>} — дублирует существующий
// файл новой строкой (замороженная копия для нового хозяина: печать видит
// только свои прикреплённые файлы, живые ссылки на чужие запрещены).
// Нет копируемого — тихий no-op {"copied": false} (первый приказ серии).
func (h *FileHandler) upload(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject]) {
	maxBody := fileMaxBytes()*4/3 + fileBodySlackBytes
	req.Body = http.MaxBytesReader(w, req.Body, maxBody)
	var up fileUpload
	if err := json.NewDecoder(req.Body).Decode(&up); err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	if up.Data == "" {
		h.copy(w, req, r, up)
		return
	}
	name := sanitizeFileName(up.FileName)
	if name == "" || name == "." || len(up.FileName) > maxFileNameLen {
		r.writeError(w, "Invalid file_name", http.StatusBadRequest)
		return
	}
	raw, err := base64.StdEncoding.DecodeString(up.Data)
	if err != nil {
		r.writeError(w, "Invalid base64 data", http.StatusBadRequest)
		return
	}
	if int64(len(raw)) == 0 || int64(len(raw)) > fileMaxBytes() {
		r.writeError(w, fmt.Sprintf("File size 1..%d bytes required", fileMaxBytes()), http.StatusBadRequest)
		return
	}
	mime, ok := fileMimeByMagic(raw)
	if !ok {
		r.writeError(w, "File must be png|jpg|pdf (magic mismatch)", http.StatusBadRequest)
		return
	}
	row := FileObject{FileName: name, Mime: mime, Size: int64(len(raw)), Data: raw}
	if err := h.db.Table(row.TableName()).Create(&row).Error; err != nil {
		r.writeError(w, "Failed to store file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	r.writeJSONResponse(w, fileMeta{
		Id: row.Id, FileName: row.FileName, Mime: row.Mime,
		Size: row.Size, IdDocument: nil, CreatedAt: row.CreatedAt,
	}, http.StatusCreated)
}

// copy дублирует файл новой строкой (тот же байты/имя/mime).
// id_document в запросе — сразу прикрепить копию (проверяется существование
// документа; чужие файлы копировать можно — это копия, не угон).
func (h *FileHandler) copy(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject], up fileUpload) {
	srcID, ok := printID64(up.CopyFrom)
	if !ok {
		// Некого копировать (первый приказ / нет прошлого логотипа) — no-op.
		r.writeJSONResponse(w, map[string]any{"copied": false}, http.StatusOK)
		return
	}
	var src FileObject
	if err := h.db.Table(src.TableName()).Where("id = ?", srcID).First(&src).Error; err != nil {
		r.writeError(w, "Source file not found", http.StatusNotFound)
		return
	}
	row := FileObject{FileName: src.FileName, Mime: src.Mime, Size: src.Size, Data: src.Data}
	if up.IdDocument != nil {
		if *up.IdDocument <= 0 {
			r.writeError(w, "Invalid id_document", http.StatusBadRequest)
			return
		}
		var exists int64
		if err := h.db.Table(h.parentTable).Where("id = ?", *up.IdDocument).Count(&exists).Error; err != nil {
			r.writeError(w, "Failed to check document: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if exists == 0 {
			r.writeError(w, "Document not found", http.StatusBadRequest)
			return
		}
		row.IdDocument = up.IdDocument
	}
	if err := h.db.Table(row.TableName()).Create(&row).Error; err != nil {
		r.writeError(w, "Failed to copy file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	r.writeJSONResponse(w, map[string]any{
		"id": row.Id, "file_name": row.FileName, "mime": row.Mime,
		"size": row.Size, "id_document": row.IdDocument, "copied": true,
	}, http.StatusCreated)
}

// list отдаёт МЕТАДАННЫЕ файлов документа (байтов нет никогда).
func (h *FileHandler) list(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject]) {
	idDoc := strings.TrimSpace(req.URL.Query().Get("id_document"))
	if idDoc == "" {
		r.writeError(w, "Missing id_document parameter", http.StatusBadRequest)
		return
	}
	docID, err := strconv.ParseInt(idDoc, 10, 64)
	if err != nil || docID <= 0 {
		r.writeError(w, "Invalid id_document parameter", http.StatusBadRequest)
		return
	}
	var rows []FileObject
	if err := h.db.Table(FileObject{}.TableName()).
		Select("id, file_name, mime, size, id_document, created_at").
		Where("id_document = ?", docID).Order("id").Find(&rows).Error; err != nil {
		r.writeError(w, "Failed to list files: "+err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]fileMeta, 0, len(rows))
	for _, row := range rows {
		out = append(out, fileMeta{
			Id: row.Id, FileName: row.FileName, Mime: row.Mime,
			Size: row.Size, IdDocument: row.IdDocument, CreatedAt: row.CreatedAt,
		})
	}
	r.writeJSONResponse(w, map[string]any{"data": out}, http.StatusOK)
}

// content отдаёт байты одного файла.
func (h *FileHandler) content(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject], id string) {
	fileID, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil || fileID <= 0 {
		r.writeError(w, "Invalid content parameter", http.StatusBadRequest)
		return
	}
	var row FileObject
	if err := h.db.Table(row.TableName()).
		Select("file_name, mime, size, data").Where("id = ?", fileID).First(&row).Error; err != nil {
		r.writeError(w, "File not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", row.Mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeFileName(row.FileName)))
	w.Header().Set("Content-Length", strconv.FormatInt(row.Size, 10))
	_, _ = w.Write(row.Data)
}

// attach привязывает сироту к документу (шаг пайплайна сохранения).
// Правила: привязать можно только сироту (чужой файл увести нельзя);
// повторная привязка к тому же документу — тихий no-op (re-save);
// replace — сначала отцепить остальные файлы документа (смена одиночного
// логотипа; флаг едет в теле шага, т.к. query пайплайн строит только из
// filter, а тело — из map); id_document=null — отцепить (станет сиротой).
// id берётся из query (?id=, прямой вызов) или из тела (шаг пайплайна без
// фильтра: map {"id": "id_logo"}). Отсутствующий/битый id без явного ?id=
// (плейсхолдер без значения в контексте) — тихий no-op 200, а не 400:
// шаг привязки обязан молча пропускаться, когда файла нет.
func (h *FileHandler) attach(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject]) {
	var body struct {
		Id         any    `json:"id"`
		IdDocument *int64 `json:"id_document"`
		Replace    bool   `json:"replace"`
	}
	// Тело — крошечный JSON привязки (id/id_document/replace); кап как у PUT,
	// чтобы безлимитный Decode не держал соединение (там же MaxBytesReader).
	req.Body = http.MaxBytesReader(w, req.Body, attachMaxBodyBytes)
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()
	replace := req.URL.Query().Get("replace") != "" || body.Replace
	fileID, ok := printID64(body.Id)
	if !ok {
		if idStr := strings.TrimSpace(req.URL.Query().Get("id")); idStr != "" {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil && id > 0 {
				fileID, ok = id, true
			} else {
				r.writeError(w, "Invalid id parameter", http.StatusBadRequest)
				return
			}
		}
	}
	if !ok {
		// Нет id ни в query, ни в теле (поле файла не заполнено) —
		// нечего привязывать, тихий no-op для пайплайна.
		r.writeJSONResponse(w, map[string]any{"attached": false}, http.StatusOK)
		return
	}
	var row FileObject
	if err := h.db.Table(row.TableName()).Where("id = ?", fileID).First(&row).Error; err != nil {
		r.writeError(w, "File not found", http.StatusNotFound)
		return
	}
	// Отцепление — всегда разрешено (файл станет сиротой).
	if body.IdDocument == nil {
		if err := h.db.Table(row.TableName()).Where("id = ?", fileID).
			Update("id_document", nil).Error; err != nil {
			r.writeError(w, "Failed to detach file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		r.writeJSONResponse(w, map[string]any{"id": fileID, "id_document": nil}, http.StatusOK)
		return
	}
	docID := *body.IdDocument
	if docID <= 0 {
		r.writeError(w, "Invalid id_document", http.StatusBadRequest)
		return
	}
	// Уже привязан к тому же документу — no-op (идемпотентный re-save).
	if row.IdDocument != nil && *row.IdDocument == docID {
		r.writeJSONResponse(w, map[string]any{"id": fileID, "id_document": docID}, http.StatusOK)
		return
	}
	// Привязан к чужому документу — отказ (угон невозможен).
	if row.IdDocument != nil {
		r.writeError(w, "File is attached to another document", http.StatusConflict)
		return
	}
	// Документ-хозяин обязан существовать (иначе — будущий труп для GC).
	var exists int64
	if err := h.db.Table(h.parentTable).Where("id = ?", docID).Count(&exists).Error; err != nil {
		r.writeError(w, "Failed to check document: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if exists == 0 {
		r.writeError(w, "Document not found", http.StatusBadRequest)
		return
	}
	tx := h.db.Begin()
	if tx.Error != nil {
		r.writeError(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	// Смена одиночного файла: старые привязки документа сбросить.
	if replace {
		if err := tx.Table(row.TableName()).Where("id_document = ? AND id <> ?", docID, fileID).
			Update("id_document", nil).Error; err != nil {
			tx.Rollback()
			r.writeError(w, "Failed to replace file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Table(row.TableName()).Where("id = ?", fileID).
		Update("id_document", docID).Error; err != nil {
		tx.Rollback()
		r.writeError(w, "Failed to attach file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit().Error; err != nil {
		r.writeError(w, "Failed to commit attach: "+err.Error(), http.StatusInternalServerError)
		return
	}
	r.writeJSONResponse(w, map[string]any{"id": fileID, "id_document": docID}, http.StatusOK)
}

// remove удаляет строку файла (?id=, 404 если нет) или все файлы документа
// (?id_document= — фильтрованный multi-DELETE для шага on_delete сущности;
// ноль совпадений — 204, как у generic-DELETE; пустой DELETE — 400).
func (h *FileHandler) remove(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject]) {
	q := req.URL.Query()
	if idStr := strings.TrimSpace(q.Get("id")); idStr != "" {
		fileID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || fileID <= 0 {
			r.writeError(w, "Invalid id parameter", http.StatusBadRequest)
			return
		}
		res := h.db.Table(FileObject{}.TableName()).Where("id = ?", fileID).Delete(&FileObject{})
		if res.Error != nil {
			r.writeError(w, "Failed to delete file: "+res.Error.Error(), http.StatusInternalServerError)
			return
		}
		if res.RowsAffected == 0 {
			r.writeError(w, "File not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if docStr := strings.TrimSpace(q.Get("id_document")); docStr != "" {
		docID, err := strconv.ParseInt(docStr, 10, 64)
		if err != nil || docID <= 0 {
			r.writeError(w, "Invalid id_document parameter", http.StatusBadRequest)
			return
		}
		if err := h.db.Table(FileObject{}.TableName()).Where("id_document = ?", docID).
			Delete(&FileObject{}).Error; err != nil {
			r.writeError(w, "Failed to delete files: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r.writeError(w, "Missing id parameter", http.StatusBadRequest)
}

// CollectGarbage удаляет сирот старше ttl: с пустым id_document, а также
// с номером несуществующего документа (JOIN с основной таблицей).
// Возвращает число удалённых строк.
func (h *FileHandler) CollectGarbage(ttl time.Duration) (int64, error) {
	// parentTable — из конфига сервиса, но летит в сырой SQL: валидируем
	// тем же регекспом колонок (ядро — strict-first даже на своих).
	if !reFilterColumn.MatchString(h.parentTable) {
		return 0, fmt.Errorf("invalid parent table %q", h.parentTable)
	}
	cutoff := time.Now().Add(-ttl)
	res := h.db.Table(FileObject{}.TableName()).Where(
		"created_at < ? AND (id_document IS NULL OR NOT EXISTS (SELECT 1 FROM "+h.parentTable+" p WHERE p.id = "+FileObject{}.TableName()+".id_document))",
		cutoff,
	).Delete(&FileObject{})
	return res.RowsAffected, res.Error
}

// StartFileGC запускает фоновую чистку сирот (лёгкий тикер: один DELETE
// за интервал; планировщика в стенде нет, поэтому горутина в процессе).
// Останавливается вместе с процессом (канал не нужен — процесс один).
func (h *FileHandler) StartFileGC() {
	go func() {
		ttl := fileOrphanTTL()
		interval := fileGCInterval()
		log.Printf("[files] GC every %v, orphan TTL %v (table %s)", interval, ttl, h.parentTable)
		t := time.NewTicker(fileGCStartupDelay)
		defer t.Stop()
		first := true
		for range t.C {
			if first {
				// Первая чистка — через стартовую задержку, дальше по интервалу.
				first = false
				t.Reset(interval)
			}
			n, err := h.CollectGarbage(ttl)
			if err != nil {
				log.Printf("[files] GC error: %v", err)
				continue
			}
			if n > 0 {
				log.Printf("[files] GC removed %d orphan(s)", n)
			}
		}
	}()
}

// fileMetaByID читает байты файла для печати (только своего документа:
// id обязан совпадать — чужие файлы в чужую печать не текут).
func fileBytesForPrint(db *gorm.DB, fileID, docID int64) ([]byte, string, error) {
	var row FileObject
	if err := db.Table(row.TableName()).
		Select("file_name, mime, data, id_document").
		Where("id = ?", fileID).First(&row).Error; err != nil {
		return nil, "", fmt.Errorf("print file #%d not found", fileID)
	}
	if row.IdDocument == nil || *row.IdDocument != docID {
		return nil, "", fmt.Errorf("print file #%d is not attached to document #%d", fileID, docID)
	}
	if _, ok := fileMimeByMagic(row.Data); !ok {
		return nil, "", fmt.Errorf("print file #%d failed magic check", fileID)
	}
	return row.Data, row.Mime, nil
}
