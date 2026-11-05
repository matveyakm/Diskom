package restController

// HTTP-проводка печати: двухшаговая выдача на тех же роутах.
//
//	POST /api/<table>?print=pdf|docx  тело {"id": N} → 200 {token,size,filename,mime}
//	GET  /api/<table>?print=file&token=...           → байты файла (разово, TTL 5 мин)
//
// Отдельных роутов не заводим (в app.go всего 3) — ветвимся по query
// внутри Create/Get. Нет шаблона у сущности — 404 (пилот только приём).
// Битый шаблон — 500 (ошибка оператора, запечён в образ).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// handlePrintBuild собирает файл и кладёт в store, возвращает метаданные.
func (r *CRUDRepository[T]) handlePrintBuild(w http.ResponseWriter, req *http.Request) {
	format := req.URL.Query().Get("print")
	if format != "pdf" && format != "docx" {
		r.writeError(w, "Invalid print format, want pdf|docx", http.StatusBadRequest)
		return
	}

	var rawBody map[string]any
	if err := json.NewDecoder(req.Body).Decode(&rawBody); err != nil {
		r.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	idValue, present := rawBody["id"]
	if !present || idValue == nil {
		r.writeError(w, "Missing id parameter", http.StatusBadRequest)
		return
	}
	ids, err := parseIDs(fmt.Sprintf("%v", idValue))
	if err != nil || len(ids) != 1 {
		r.writeError(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}
	id := ids[0]

	tpl, found, err := loadPrintTemplate()
	if err != nil {
		r.writeError(w, "Failed to load print template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		r.writeError(w, "No print template for this entity", http.StatusNotFound)
		return
	}

	var row map[string]any
	if err := r.db.Table(r.tableName).Where("id = ?", id).Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.writeError(w, "Record not found", http.StatusNotFound)
			return
		}
		r.writeError(w, "Failed to retrieve record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Имена вместо id: подменяем до рендера (fail-open внутри).
	record := resolvePrintRefs(row, tpl, r.httpClient)

	// Обязательные поля шаблона ({{x!}}): пустое значение — честная ошибка
	// сборки, а не дыра в документе. Пользователь раньше замечает проблему
	// и зовёт администратора, чем подписывает неверный приказ.
	if err := validatePrintRow(tpl, record); err != nil {
		r.writeError(w, "Cannot print: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Картинки шаблона — заранее (битый файл — 500, операторская ошибка).
	// src: файл из папки шаблона; field: байты из таблицы files того же
	// сервиса (id файла из строки документа, принадлежность документу
	// проверяется — чужие файлы в чужую печать не текут); files.tag:
	// новейший файл ЭТОГО документа с меткой (документ своей таблицы файлов
	// не знает, указателя в строке нет). Пустое поле / нет файла с меткой —
	// картинку пропускаем (как пустой field-блок), а не рвём сборку.
	images := make(map[string][]byte)
	for _, b := range tpl.Blocks {
		if b.Type != "image" {
			continue
		}
		if _, dup := images[b.imageKey()]; dup {
			continue
		}
		if b.Src != "" {
			img, _, err := loadPrintImage(b.Src)
			if err != nil {
				r.writeError(w, "Failed to load print image: "+err.Error(), http.StatusInternalServerError)
				return
			}
			images[b.imageKey()] = img
			continue
		}
		var fileID int64
		what := ""
		if b.Field != "" {
			fid, ok := printID64(record[b.Field])
			if !ok {
				// Поле пустое — пропускаем, логотип может появиться позже.
				continue
			}
			fileID, what = fid, "field "+b.Field
		} else {
			tag := strings.TrimSpace(b.Files.Tag)
			var f FileObject
			q := r.db.Table(f.TableName()).Select("id").
				Where("id_document = ? AND tag = ?", id, tag).Order("id DESC")
			if err := q.First(&f).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					// Нет файла с меткой — пропускаем, как пустое поле.
					continue
				}
				r.writeError(w, "Failed to load print image: "+err.Error(), http.StatusInternalServerError)
				return
			}
			fileID, what = f.Id, "files.tag "+tag
		}
		img, mime, err := fileBytesForPrint(r.db, fileID, id)
		if err != nil {
			r.writeError(w, "Failed to load print image: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if mime == "application/pdf" {
			// Блок image умеет только растр (png/jpg): PDF прикрепили не туда.
			// Громкая 500 с источником вместо битого документа (философия).
			r.writeError(w, "Failed to load print image: "+what+" is PDF, want png|jpg", http.StatusInternalServerError)
			return
		}
		images[b.imageKey()] = img
	}

	var (
		data []byte
		mime string
		ext  string
	)
	switch format {
	case "pdf":
		mime, ext = printPdfMime, "pdf"
		if data, err = renderPrintPDF(tpl, record, images); err != nil {
			r.writeError(w, "Failed to render PDF: "+err.Error(), http.StatusInternalServerError)
			return
		}
	case "docx":
		mime, ext = printDocxMime, "docx"
		if data, err = renderPrintDOCX(tpl, record, images); err != nil {
			r.writeError(w, "Failed to render DOCX: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	filename := fmt.Sprintf("%s-%d.%s", r.tableName, id, ext)
	token := printTokenStore.put(printFile{Bytes: data, Filename: filename, Mime: mime})

	r.writeJSONResponse(w, printResponse{
		Token:    token,
		Size:     len(data),
		Filename: filename,
		Mime:     mime,
	}, http.StatusOK)
}

// handlePrintFile отдаёт готовый файл разово (повторный GET — 404).
func (r *CRUDRepository[T]) handlePrintFile(w http.ResponseWriter, req *http.Request) {
	token := req.URL.Query().Get("token")
	if token == "" {
		r.writeError(w, "Missing token parameter", http.StatusBadRequest)
		return
	}
	f, ok := printTokenStore.get(token)
	if !ok {
		r.writeError(w, "Print file expired or invalid token", http.StatusNotFound)
		return
	}

	disposition := "attachment"
	if f.Mime == printPdfMime {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", f.Mime)
	w.Header().Set("Content-Disposition", disposition+`; filename="`+f.Filename+`"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(f.Bytes)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(f.Bytes)
}
