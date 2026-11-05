package restController

// Файловые приложения сущностей: отдельная таблица `files` в БД каждого
// микросервиса (решение: файлы живут там же, где их хозяин).
//
// Связь: files.id_document → id строки основной таблицы (1:N, NULL =
// сирота: загружен, но ни к чему не привязан). Документ своей таблицы
// файлов НЕ знает: никаких id_file-колонок в сущностях; файловое поле формы —
// виртуальное (живёт только в контексте пайплайна, в строку не пишется).
//
// Поток (договорённость):
//  1. POST /api/files {file_name, data(base64)} → orphan-строка {id...}.
//     Отдельный запрос (файл НЕ едет внутри JSON документа).
//     Вариант {"copy_from": <id>, "id_document"?: <doc>} — дублирует файл
//     новой строкой. Нечего копировать — тихий no-op {"copied": false}.
//  2. Сохранение документа: пайплайн PUT /api/files {ids: [...],
//     id_document: <id>} — прикрепить выбранные (только сирот; чужой файл
//     увести нельзя — 409; повтор — тихий no-op, идемпотентный re-save).
//     Отдельный шаг PUT {ids: [...снятые...], id_document: null} — отцепить
//     ранее привязанные, снятые крестиком (становятся сиротами, НЕ удаляются;
//     строка files при этом не трогается). Свежие загрузки, снятые до
//     сохранения, никуда не отправляются вообще: они и так сироты.
//     Без валидных id — тихий no-op 200 (шаг обязан пропускаться, когда
//     файлов нет).
//  3. GET /api/files?id_document= — МЕТАДАННЫЕ без байтов (грид и формы).
//     GET /api/files?content= — байты с Content-Type/Disposition.
//  4. DELETE /api/files?id= — удаление строки. Привязанные файлы чистятся
//     шагом on_delete сущности (DELETE /api/files?id_document=).
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

// Меток ролей больше нет (связь — только id_document, печать с ролями
// решается отдельно, без хардкода в shared).

// parseFileIDs собирает id файлов из тела (id — один, ids — пачка):
// годные (>0) — в работу, мусор мимо. Неразрешённый плейсхолдер пайплайна
// (строка без значения в контексте) printID64 не проходит — и это пропуск
// шага выше по стеку, а не ошибка.
// Ноль валидных — не ошибка разбора, вызывающий решает (обычно no-op).
func parseFileIDs(single any, multi []any) []int64 {
	var out []int64
	seen := make(map[int64]bool)
	add := func(v any) {
		if id, ok := printID64(v); ok && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	add(single)
	for _, v := range multi {
		add(v)
	}
	return out
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
// Вариант {"copy_from": <id>, "id_document"?: <doc>} — дублирует файл новой
// строкой (чужой файл копировать можно — это копия, не угон).
// Нечего копировать (плейсхолдер без значения) — тихий no-op {"copied": false}.
// id_document в запросе — сразу прикрепить копию (документ обязан существовать).
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
	if name == "" || len(up.FileName) > maxFileNameLen {
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

// copy дублирует файл новой строкой (те же байты/имя/mime).
// id_document в запросе — сразу прикрепить копию (проверяется существование
// документа; чужие файлы копировать можно — это копия, не угон).
func (h *FileHandler) copy(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject], up fileUpload) {
	srcID, ok := printID64(up.CopyFrom)
	if !ok {
		// Некого копировать (плейсхолдер без значения в контексте) — no-op.
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
	q := h.db.Table(FileObject{}.TableName()).
		Select("id, file_name, mime, size, id_document, created_at").
		Where("id_document = ?", docID)
	var rows []FileObject
	if err := q.Order("id").Find(&rows).Error; err != nil {
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

// attach привязывает выбранные файлы к документу (шаг пайплайна сохранения).
// Тело: {ids: [...], id_document: <id>} (id — один, для совместимости). Правила:
//   - прикрепить можно только сироту (чужой файл увести нельзя — 409);
//     уже привязан к тому же документу — тихий no-op (идемпотентный re-save);
//   - replace — отцепить остальные файлы документа (становятся сиротами,
//     НЕ удаляются);
//   - id_document=null — отцепить перечисленные (снятые крестиком ранее
//     привязанные; строка files при этом не удаляется);
//   - без валидных id — тихий no-op 200, а не 400: шаг обязан молча
//     пропускаться, когда файлов нет (поле не заполнено, плейсхолдер
//     без значения в контексте).
// Документ-хозяин обязан существовать (иначе — будущий труп для GC).
func (h *FileHandler) attach(w http.ResponseWriter, req *http.Request, r *CRUDRepository[FileObject]) {
	var body struct {
		Id any `json:"id"`
		// Ids — пачка (массив из пайплайна), но принимаем и скаляр:
		// неразрешённый плейсхолдер прилетает строкой — это пропуск
		// шага (как раньше одиночный id), а не 400.
		Ids        any    `json:"ids"`
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
	// Ids нормализуем к пачке: массив как есть, скаляр — как один id
	// (строка-плейсхолдер дальше молча отфильтруется в parseFileIDs).
	var multi []any
	switch t := body.Ids.(type) {
	case nil:
	case []any:
		multi = t
	default:
		multi = []any{t}
	}
	ids := parseFileIDs(body.Id, multi)
	if len(ids) == 0 {
		if idStr := strings.TrimSpace(req.URL.Query().Get("id")); idStr != "" {
			if id, err := strconv.ParseInt(idStr, 10, 64); err == nil && id > 0 {
				ids = []int64{id}
			} else {
				r.writeError(w, "Invalid id parameter", http.StatusBadRequest)
				return
			}
		}
	}
	if len(ids) == 0 {
		// Нечего привязывать — тихий no-op для пайплайна.
		r.writeJSONResponse(w, map[string]any{"attached": false}, http.StatusOK)
		return
	}
	byID := make(map[int64]FileObject, len(ids))
	var rows []FileObject
	if err := h.db.Table(FileObject{}.TableName()).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		r.writeError(w, "Failed to find files: "+err.Error(), http.StatusInternalServerError)
		return
	}
	for _, row := range rows {
		byID[row.Id] = row
	}
	for _, id := range ids {
		if _, ok := byID[id]; !ok {
			r.writeError(w, "File not found", http.StatusNotFound)
			return
		}
	}
	// Отцепление ранее привязанных, снятых крестиком (файлы станут сиротами,
	// строка files не удаляется — подберёт GC, если файл так и не вернут).
	if body.IdDocument == nil {
		if err := h.db.Table(FileObject{}.TableName()).Where("id IN ?", ids).
			Update("id_document", nil).Error; err != nil {
			r.writeError(w, "Failed to detach files: "+err.Error(), http.StatusInternalServerError)
			return
		}
		r.writeJSONResponse(w, map[string]any{"ids": ids, "id_document": nil}, http.StatusOK)
		return
	}
	docID := *body.IdDocument
	if docID <= 0 {
		r.writeError(w, "Invalid id_document", http.StatusBadRequest)
		return
	}
	// Привязан к чужому документу — отказ (угон невозможен).
	for _, id := range ids {
		if row := byID[id]; row.IdDocument != nil && *row.IdDocument != docID {
			r.writeError(w, "File is attached to another document", http.StatusConflict)
			return
		}
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
	// Точное множество: файлы документа вне выборки — в сироты.
	if replace {
		q := tx.Table(FileObject{}.TableName()).Where("id_document = ? AND id NOT IN ?", docID, ids)
		if err := q.Update("id_document", nil).Error; err != nil {
			tx.Rollback()
			r.writeError(w, "Failed to replace files: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, id := range ids {
		if err := tx.Table(FileObject{}.TableName()).Where("id = ?", id).
			Update("id_document", docID).Error; err != nil {
			tx.Rollback()
			r.writeError(w, "Failed to attach file: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		r.writeError(w, "Failed to commit attach: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp := map[string]any{"ids": ids, "id_document": docID}
	if len(ids) == 1 {
		resp["id"] = ids[0]
	}
	r.writeJSONResponse(w, resp, http.StatusOK)
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
