package print

// Filler слота "career": построчный текст интервалов должностей за период:
// ФИО, затем строки «- В период с … по … являлся/является <должность> <отдел>».
//
// Имена берутся из регистров-таймлайнов на дату T, где T = max(ddate отчёта,
// date_to): отчёт показывает имена, актуальные не раньше конца своего периода,
// но замороженные параметрами запуска (перепечать даёт тот же текст).
// max, а не ddate: иначе найм после создания отчёта в будущий период
// ронял бы печать громкой ошибкой вместо показа интервала.
// Регистры: employee_data_ledger (ФИО), job_title_data_ledger (должность,
// творительный падеж), department_data_ledger (отдел, родительный падеж).
// События staff_placement_ledger спариваются FIFO по тройке id
// (сотрудник, отдел, должность): open открывает интервал, close закрывает
// самый старый открытый. Параллельные позиции на разных тройках живут
// независимыми очередями; повторное назначение на ту же тройку без
// снятия — глубиной очереди 2. Остаток не считается (не численность).
// Сирота-close, неизвестный event, нулевая тройка, отсутствие имени
// на дату T — громкая ошибка: неверные данные хуже отсутствующего отчёта.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"shared/report"
)

// Slots — реестр заполнителей сервиса: дырка → функция (связь совпадением
// имён; смысл задаёт позиция блока в шаблоне).
var Slots = report.Slots{"career": careerSlot, "emblem": emblemSlot}

// tenure — один интервал позиции: [from, to), to == nil — открыт.
type tenure struct {
	emp, dep, job int64
	from          time.Time
	to            *time.Time
}

// names — отображаемые имена на дату формирования отчёта.
type names struct {
	fio map[int64]string // сотрудник → ФИО (именительный)
	job map[int64]string // должность → название (творительный)
	dep map[int64]string // отдел → название (родительный)
}

// resolveNames — имена на дату T из регистров-таймлайнов: последнее значение
// с ddate ≤ T по каждой сущности. Порядок берётся из самих данных
// (сортировка (день, id) на нашей стороне — от серверного порядка не зависим).
// Пустое fio/имя — как отсутствие (громко на этапе использования).
func resolveNames(ctx report.Ctx, t time.Time) (names, error) {
	n := names{fio: map[int64]string{}, job: map[int64]string{}, dep: map[int64]string{}}
	empRows, err := ctx.Fetch("employee_data_ledger", map[string]string{
		"ddate__lte": t.Format(time.RFC3339),
		"sort":       "ddate",
		"order":      "asc",
	})
	if err != nil {
		return n, fmt.Errorf("career: employee_data: %w", err)
	}
	ordered, err := orderByDayID(empRows)
	if err != nil {
		return n, fmt.Errorf("career: employee_data: %w", err)
	}
	for _, r := range ordered {
		id, _ := report.Int(r["id_employee"])
		if id == 0 {
			return n, fmt.Errorf("career: employee_data row %v: bad id", r["id"])
		}
		d, _ := careerDay(r["ddate"])
		if d.After(t) {
			continue
		}
		if s, _ := r["fio"].(string); s != "" {
			n.fio[id] = s
		}
	}
	jobRows, err := ctx.Fetch("job_title_data_ledger", map[string]string{
		"ddate__lte": t.Format(time.RFC3339),
		"sort":       "ddate",
		"order":      "asc",
	})
	if err != nil {
		return n, fmt.Errorf("career: job_title_data: %w", err)
	}
	ordered, err = orderByDayID(jobRows)
	if err != nil {
		return n, fmt.Errorf("career: job_title_data: %w", err)
	}
	for _, r := range ordered {
		id, _ := report.Int(r["id_job_title"])
		if id == 0 {
			return n, fmt.Errorf("career: job_title_data row %v: bad id", r["id"])
		}
		if s, _ := r["name_instrumental"].(string); s != "" {
			n.job[id] = s
		}
	}
	depRows, err := ctx.Fetch("department_data_ledger", map[string]string{
		"ddate__lte": t.Format(time.RFC3339),
		"sort":       "ddate",
		"order":      "asc",
	})
	if err != nil {
		return n, fmt.Errorf("career: department_data: %w", err)
	}
	ordered, err = orderByDayID(depRows)
	if err != nil {
		return n, fmt.Errorf("career: department_data: %w", err)
	}
	for _, r := range ordered {
		id, _ := report.Int(r["id_department"])
		if id == 0 {
			return n, fmt.Errorf("career: department_data row %v: bad id", r["id"])
		}
		if s, _ := r["name_genitive"].(string); s != "" {
			n.dep[id] = s
		}
	}
	return n, nil
}

// orderByDayID — копия строк, отсортированная по (день, id): победитель
// last-wins детерминирован независимо от порядка выдачи сервера.
// Битые даты — громкая ошибка с id строки (проверка после сортировки,
// чтобы порядок обхода ошибок тоже был детерминирован).
func orderByDayID(rows []map[string]any) ([]map[string]any, error) {
	ordered := make([]map[string]any, 0, len(rows))
	ordered = append(ordered, rows...)
	sort.SliceStable(ordered, func(i, j int) bool {
		di, erri := careerDay(ordered[i]["ddate"])
		dj, errj := careerDay(ordered[j]["ddate"])
		if erri != nil || errj != nil {
			return erri == nil
		}
		if !di.Equal(dj) {
			return di.Before(dj)
		}
		ii, _ := report.Int(ordered[i]["id"])
		ji, _ := report.Int(ordered[j]["id"])
		return ii < ji
	})
	for _, r := range ordered {
		if _, err := careerDay(r["ddate"]); err != nil {
			return nil, fmt.Errorf("row %v: bad date: %w", r["id"], err)
		}
	}
	return ordered, nil
}

// careerSlot — построчный текст «как в приказе»: ФИО, затем строки
// «- В период с … по … являлся/является <должность-Т> <отдел-Р>».
// Источник — ТОЛЬКО регистры (никаких справочных забегов и документов):
//
//	staff_placement_ledger — события open/close (тройка id — ключ спаривания);
//	employee_data_ledger — таймлайн ФИО («ФИО на дату T» = последнее fio
//	  сотрудника с ddate ≤ T, где T — ddate отчёта, дата его формирования);
//	job_title_data_ledger / department_data_ledger — таймлайны названий
//	  («название на дату T» — последняя строка с ddate ≤ T).
//
// События спариваются FIFO по тройке (сотрудник, отдел, должность).
// Сирота-close, неизвестный event, нулевая тройка, отсутствие ФИО/названия
// на дату T — громкая ошибка: неверные данные хуже отсутствующего отчёта.
func careerSlot(ctx report.Ctx, _ report.Params) (any, error) {
	rec := ctx.Record()
	from, err := careerDay(rec["date_from"])
	if err != nil {
		return nil, fmt.Errorf("career: date_from: %w", err)
	}
	to, err := careerDay(rec["date_to"])
	if err != nil {
		return nil, fmt.Errorf("career: date_to: %w", err)
	}
	if from.After(to) {
		return nil, fmt.Errorf("career: date_from %s позже date_to %s",
			from.Format("02.01.2006"), to.Format("02.01.2006"))
	}
	events, err := ctx.Fetch("staff_placement_ledger", map[string]string{
		"ddate__lte": to.Format(time.RFC3339),
		"sort":       "ddate",
		"order":      "asc",
	})
	if err != nil {
		return nil, fmt.Errorf("career: ledger: %w", err)
	}
	// T — max(ddate строки, date_to): имена актуальны не раньше конца
	// периода, но заморожены параметрами запуска (см. шапку файла).
	formed, err := careerDay(rec["ddate"])
	if err != nil {
		return nil, fmt.Errorf("career: ddate: %w", err)
	}
	if to.After(formed) {
		formed = to
	}
	names, err := resolveNames(ctx, formed)
	if err != nil {
		return nil, err
	}
	tenures, err := pairEvents(events)
	if err != nil {
		return nil, err
	}
	type line struct {
		person string
		from   time.Time
		text   string
	}
	var lines []line
	for _, t := range tenures {
		if t.to != nil && !t.to.After(from) {
			continue // закрыт до начала периода
		}
		if t.from.After(to) {
			continue // открыт после конца периода
		}
		start := t.from
		if start.Before(from) {
			start = from
		}
		if start.After(to) {
			continue
		}
		end := t.to
		if end != nil && end.After(to) {
			end = nil // на дату отчёта ещё открыт
		}
		person, ok := names.fio[t.emp]
		if !ok {
			return nil, fmt.Errorf("career: нет ФИО сотрудника %d на %s", t.emp, formed.Format("02.01.2006"))
		}
		position, ok := names.job[t.job]
		if !ok {
			return nil, fmt.Errorf("career: нет названия должности %d на %s", t.job, formed.Format("02.01.2006"))
		}
		department, ok := names.dep[t.dep]
		if !ok {
			return nil, fmt.Errorf("career: нет названия отдела %d на %s", t.dep, formed.Format("02.01.2006"))
		}
		var text string
		if end == nil {
			text = fmt.Sprintf("- В период с %s по настоящее время является %s %s",
				start.Format("02.01.2006"), position, department)
		} else {
			text = fmt.Sprintf("- В период с %s по %s являлся %s %s",
				start.Format("02.01.2006"), end.Format("02.01.2006"), position, department)
		}
		lines = append(lines, line{person: person, from: start, text: text})
	}
	sort.SliceStable(lines, func(i, j int) bool {
		if lines[i].person != lines[j].person {
			return lines[i].person < lines[j].person
		}
		return lines[i].from.Before(lines[j].from)
	})
	var sb strings.Builder
	last := ""
	for _, l := range lines {
		if l.person != last {
			if last != "" {
				sb.WriteString("\n")
			}
			sb.WriteString(l.person)
			sb.WriteString("\n")
			last = l.person
		}
		sb.WriteString(l.text)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}

// pairEvents — FIFO-спаривание open/close по тройке. Порядок: (день, id).
func pairEvents(rows []map[string]any) ([]tenure, error) {
	ordered, err := orderByDayID(rows)
	if err != nil {
		return nil, fmt.Errorf("career: %w", err)
	}
	open := make(map[string][]tenure)
	var out []tenure
	for _, r := range ordered {
		id, _ := report.Int(r["id"])
		ev, _ := r["event"].(string)
		emp, _ := report.Int(r["id_employee"])
		dep, _ := report.Int(r["id_department"])
		job, _ := report.Int(r["id_job_title"])
		if emp == 0 || dep == 0 || job == 0 {
			return nil, fmt.Errorf("career: row %d: нулевой id тройки (сотрудник/отдел/должность)", id)
		}
		d, err := careerDay(r["ddate"])
		if err != nil {
			return nil, fmt.Errorf("career: row %d: bad date: %w", id, err)
		}
		key := fmt.Sprintf("%d|%d|%d", emp, dep, job)
		switch ev {
		case "open":
			open[key] = append(open[key], tenure{emp: emp, dep: dep, job: job, from: d})
		case "close":
			q := open[key]
			if len(q) == 0 {
				return nil, fmt.Errorf("career: row %d: close без open (сотрудник %d, отдел %d, должность %d)", id, emp, dep, job)
			}
			t := q[0]
			open[key] = q[1:]
			if d.Before(t.from) {
				return nil, fmt.Errorf("career: row %d: close %s раньше open %s", id, d.Format("02.01.2006"), t.from.Format("02.01.2006"))
			}
			t.to = &d
			out = append(out, t)
		default:
			return nil, fmt.Errorf("career: row %d: неизвестное событие %q, want open|close", id, ev)
		}
	}
	for _, q := range open {
		out = append(out, q...)
	}
	return out, nil
}

// careerDay приводит значение даты строки к дню (UTC, без времени):
// time.Time из драйвера, RFC3339 или YYYY-MM-DD строкой.
func careerDay(v any) (time.Time, error) {
	if t, ok := v.(time.Time); ok {
		if t.IsZero() {
			return time.Time{}, fmt.Errorf("empty date")
		}
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
	}
	if s, ok := v.(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			y, m, d := t.Date()
			return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
		}
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t, nil
		}
		if t, err := time.Parse("02.01.2006", s); err == nil {
			return t, nil
		}
		return time.Time{}, fmt.Errorf("bad date %q", s)
	}
	return time.Time{}, fmt.Errorf("bad date %v", v)
}
