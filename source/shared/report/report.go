// Package report — глаголы фреймворка сборки отчётов для индивидуального кода
// сервисов. Разделение (зафиксировано): сервис знает ЧТО взять и как посчитать,
// каркас знает КАК сходить и подставить. Здесь нет ни document-семантики,
// ни пер-сервисных веток: слоты адресуются plain-именами, значения —
// байты/строки/строки-таблицы. Ошибки глаголов уже контекстуализированы
// (что и где упало); фреймворк сверху дописывает только имя слота.
// Немой пропуск запрещён везде: нет данных — громкая ошибка, а не пустота.
package report

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Params — статические параметры дырки из шаблона (данные, не код).
// Например пороги бакетов диаграммы: {"a": "3", "b": "10"}.
type Params map[string]string

// FileMeta — метаданные файла без байтов (как в files list).
type FileMeta struct {
	ID   int64
	Name string
	Mime string
	Size int64
}

// Ctx — всё, что доступно заполнителю: строка документа и глаголы сходить
// за данными. Реализация — в shared/controller (там http-клиент и REF_*).
type Ctx interface {
	// Record возвращает строку документа (сырая, до ref-подмен).
	Record() map[string]any
	// Fetch — все строки list-эндпоинта (проход по страницам до конца).
	Fetch(ref string, q map[string]string) ([]map[string]any, error)
	// FetchOne — ровно одна строка (single-объект или data[0] пачки).
	// Ноль или несколько — ошибка.
	FetchOne(ref string, q map[string]string) (map[string]any, error)
	// FilesOf — метаданные файлов документа чужого сервиса.
	FilesOf(ref string, docID int64) ([]FileMeta, error)
	// FileBytes — байты одного файла чужого сервиса (+mime из заголовка).
	FileBytes(ref string, fileID, docID int64) ([]byte, string, error)
}

// SlotFunc — заполнитель одной дырки: имя дырки связано с функцией
// регистрацией (совпадение строк, смысл — только в позиции в шаблоне).
// Возврат: []byte (картинка), string (текст — text-слоты),
// []map[string]any (строки таблицы — table-слоты).
type SlotFunc func(Ctx, Params) (any, error)

// Slots — реестр заполнителей сервиса: дырка → функция.
// Регистрирует main.go сервиса одной строкой; shared документ не смотрит.
type Slots map[string]SlotFunc

// Str приводит значение строки к строке для query-параметров:
// time.Time → UTC RFC3339, целое число → без decimals, остальное — как есть.
func Str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case float32:
		return Str(float64(t))
	case int:
		return strconv.Itoa(t)
	case int8:
		return strconv.FormatInt(int64(t), 10)
	case int16:
		return strconv.FormatInt(int64(t), 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case uint:
		return strconv.FormatUint(uint64(t), 10)
	case uint8:
		return strconv.FormatUint(uint64(t), 10)
	case uint16:
		return strconv.FormatUint(uint64(t), 10)
	case uint32:
		return strconv.FormatUint(uint64(t), 10)
	case uint64:
		return strconv.FormatUint(t, 10)
	case time.Time:
		return t.UTC().Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Int приводит значение строки к id: целые числа и строки из цифр.
// Нечисловое/неположительное — false (идти не с чем — решает вызывающий).
func Int(v any) (int64, bool) {
	switch t := v.(type) {
	case int:
		if t > 0 {
			return int64(t), true
		}
	case int8:
		if t > 0 {
			return int64(t), true
		}
	case int16:
		if t > 0 {
			return int64(t), true
		}
	case int32:
		if t > 0 {
			return int64(t), true
		}
	case int64:
		if t > 0 {
			return t, true
		}
	case uint:
		if t > 0 {
			return int64(t), true
		}
	case uint8:
		if t > 0 {
			return int64(t), true
		}
	case uint16:
		if t > 0 {
			return int64(t), true
		}
	case uint32:
		if t > 0 {
			return int64(t), true
		}
	case uint64:
		if t > 0 && t <= uint64(^uint64(0)>>1) {
			return int64(t), true
		}
	case float64:
		if t == float64(int64(t)) && t > 0 {
			return int64(t), true
		}
	case float32:
		return Int(float64(t))
	case string:
		if id, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil && id > 0 {
			return id, true
		}
	}
	return 0, false
}

// ActualFileBytes — цепочка «актуальная-на-дату версия → её файл → байты»
// одним глаголом: ?actual=<field>&at=<at> + соседние фильтры, ровно один файл
// у версии (0 или 2+ — громкая ошибка: неоднозначность хуже отсутствия),
// PDF — громкая ошибка (блок image умеет только растр).
func ActualFileBytes(ctx Ctx, ref, actualField, at string, filters map[string]string) ([]byte, error) {
	q := map[string]string{"actual": actualField, "at": at}
	for k, v := range filters {
		q[k] = v
	}
	ver, err := ctx.FetchOne(ref, q)
	if err != nil {
		return nil, fmt.Errorf("actual %s as of %s: %w", ref, at, err)
	}
	verID, ok := Int(ver["id"])
	if !ok {
		return nil, fmt.Errorf("actual %s as of %s: no id in version row", ref, at)
	}
	files, err := ctx.FilesOf(ref, verID)
	if err != nil {
		return nil, fmt.Errorf("actual %s as of %s: %w", ref, at, err)
	}
	if len(files) != 1 {
		return nil, fmt.Errorf("actual %s as of %s: files %d, want 1", ref, at, len(files))
	}
	data, mime, err := ctx.FileBytes(ref, files[0].ID, verID)
	if err != nil {
		return nil, fmt.Errorf("actual %s as of %s: %w", ref, at, err)
	}
	if mime == "application/pdf" {
		return nil, fmt.Errorf("actual %s as of %s: file #%d is PDF, want png|jpg", ref, at, files[0].ID)
	}
	return data, nil
}
