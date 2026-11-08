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

	// Сборка отчёта ДО refs/validate: ветки if вычисляются по сырой строке,
	// слоты заполняются лениво — только взятой ветки. Шаблон без if/slot
	// идёт старым путём побайтово так же, как раньше.
	plainTpl := tpl
	var slotImages map[string][]byte
	var problems []PrintProblem
	if templateNeedsAssembly(tpl) {
		var merr error
		plainTpl, slotImages, problems, merr = r.materializePrint(tpl, row, r.ReportSlots)
		if merr != nil {
			r.writeError(w, "Cannot assemble report: "+merr.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Имена вместо id: подменяем до рендера (fail-open внутри).
	record := resolvePrintRefs(row, plainTpl, r.httpClient)

	// Строгие плейсхолдеры ({{x!}}): пустое значение — несмертельная проблема
	// (красная ОШИБКА в DOCX, список в PDF), а не аварийный останов.
	for _, p := range collectPrintProblems(plainTpl, record) {
		problems = append(problems, p)
	}

	// Картинки шаблона — заранее (битый файл — 500, операторская ошибка).
	// src: файл из папки шаблона; field: байты из таблицы files того же
	// сервиса (id файла из строки документа, принадлежность документу
	// проверяется — чужие файлы в чужую печать не текут); slot: байты уже
	// лежат в slotImages (положил materialize через filler сервиса).
	images := slotImages
	if images == nil {
		images = make(map[string][]byte)
	}
	for _, b := range plainTpl.Blocks {
		if b.Type != "image" {
			continue
		}
		if _, dup := images[b.imageKey()]; dup {
			continue
		}
		if b.Slot != "" {
			// Байты обязан был положить materialize; их отсутствие —
			// баг сборки, а не повод молча пропустить картинку.
			if _, ok := images[b.imageKey()]; !ok {
				r.writeError(w, "Failed to load print image: slot "+b.Slot+" has no bytes", http.StatusInternalServerError)
				return
			}
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
		fileID, ok := printID64(record[b.Field])
		if !ok {
			// Поле пустое — картинку пропускаем (как пустой field-блок),
			// а не рвём сборку: логотип может появиться позже.
			continue
		}
		img, _, err := fileBytesForPrint(r.db, fileID, id)
		if err != nil {
			// Данные битые/нет (включая чужой файл и PDF вместо картинки) —
			// не роняем: чёрный квадрат в DOCX, список в PDF (см. ниже).
			// Решение пользователя: устойчивость важнее строгости,
			// недостающее допишет руками поверх квадрата.
			label := b.Field
			if l := printFieldLabels(plainTpl)[b.Field]; l != "" {
				label = l
			}
			problems = append(problems, PrintProblem{
				Kind:    "image",
				Ref:     b.Field,
				Label:   label,
				Message: err.Error(),
			})
			images[b.imageKey()] = printFallbackSquare()
			continue
		}
		images[b.imageKey()] = img
	}

	// PDF править руками нельзя — при проблемах роняем сборку списком,
	// а не молчаливым неверным документом. DOCX идёт на рендер как есть:
	// красные ОШИБКА и чёрные квадраты уже на местах.
	if format == "pdf" && len(problems) > 0 {
		r.writePrintProblems(w, format, problems)
		return
	}

	var (
		data []byte
		mime string
		ext  string
	)
	switch format {
	case "pdf":
		mime, ext = printPdfMime, "pdf"
		if data, err = renderPrintPDF(plainTpl, record, images); err != nil {
			r.writeError(w, "Failed to render PDF: "+err.Error(), http.StatusInternalServerError)
			return
		}
	case "docx":
		mime, ext = printDocxMime, "docx"
		if data, err = renderPrintDOCX(plainTpl, record, images); err != nil {
			r.writeError(w, "Failed to render DOCX: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	filename := fmt.Sprintf("%s-%d.%s", r.tableName, id, ext)
	token := printTokenStore.put(printFile{Bytes: data, Filename: filename, Mime: mime})

	resp := printResponse{
		Token:    token,
		Size:     len(data),
		Filename: filename,
		Mime:     mime,
	}
	// DOCX с заглушками: файл отдан, но предупредить надо — список едет
	// рядом с токеном, фронт показывает модалку-предупреждение, скачивание
	// при этом доступно.
	if format == "docx" && len(problems) > 0 {
		resp.Warnings = problems
	}
	r.writeJSONResponse(w, resp, http.StatusOK)
}

// writePrintProblems отвечает 500 со списком несмертельных проблем сборки.
// Только PDF-путь: PDF править руками нельзя, поэтому вместо молчаливого
// неверного документа — честный список. DOCX вместо этого подменяет
// заглушки (красная ОШИБКА, чёрный квадрат) и отдаёт 200.
func (r *CRUDRepository[T]) writePrintProblems(w http.ResponseWriter, format string, problems []PrintProblem) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	msg := fmt.Sprintf("print %s blocked by %d data problem(s)", format, len(problems))
	_ = json.NewEncoder(w).Encode(map[string]any{"error": msg, "details": problems})
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
