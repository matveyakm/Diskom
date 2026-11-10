package print

// Filler слота "career": интервалы должностей сотрудников за период.
//
// Источник — ТОЛЬКО staff_placement_ledger (один Fetch, никаких забегов по
// справочникам и документам: отображаемые имена лежат снапшотами в самих
// строках регистра — employee_name, job_title_instrumental,
// department_genitive). События спариваются FIFO по тройке
// (сотрудник, отдел, должность): open открывает интервал, close закрывает
// самый старый открытый. Параллельные позиции на разных тройках живут
// независимыми очередями; повторное назначение на ту же тройку без
// снятия — глубиной очереди 2.
// Остаток НЕ считается (этот отчёт — не про численность): накопления нет,
// колонки balance в регистре нет.
// Сирота-close (закрытие без открытия) и неизвестный event — громкая
// ошибка: неверные данные хуже отсутствующего отчёта.

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"shared/report"
)

// Slots — реестр заполнителей сервиса: дырка → функция (связь совпадением
// имён; смысл задаёт позиция картинки/таблицы в шаблоне).
var Slots = report.Slots{"career": careerSlot, "emblem": emblemSlot}

// tenure — один интервал позиции: [from, to), to == nil — открыт.
type tenure struct {
	person     string
	position   string // должность, творительный падеж (снапшот)
	department string // отдел, родительный падеж (снапшот)
	from       time.Time
	to         *time.Time
}

// careerSlot — построчный текст «как в приказе»: ФИО, затем строки
// «- В период с … по … являлся/является <должность-Т> <отдел-Р>».
// Источник — ТОЛЬКО staff_placement_ledger (один Fetch, никаких забегов по
// справочникам и документам: отображаемые имена лежат снапшотами в самих
// строках регистра — employee_name, job_title_instrumental,
// department_genitive). События спариваются FIFO по тройке
// (сотрудник, отдел, должность): open открывает интервал, close закрывает
// самый старый открытый. Параллельные позиции на разных тройках живут
// независимыми очередями; повторное назначение на ту же тройку без
// снятия — глубиной очереди 2.
// Остаток НЕ считается (этот отчёт — не про численность): накопления нет,
// колонки balance в регистре нет.
// Сирота-close (закрытие без открытия) и неизвестный event — громкая
// ошибка: неверные данные хуже отсутствующего отчёта.
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
		var text string
		if end == nil {
			text = fmt.Sprintf("- В период с %s по настоящее время является %s %s",
				start.Format("02.01.2006"), t.position, t.department)
		} else {
			text = fmt.Sprintf("- В период с %s по %s являлся %s %s",
				start.Format("02.01.2006"), end.Format("02.01.2006"), t.position, t.department)
		}
		lines = append(lines, line{person: t.person, from: start, text: text})
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
		person, _ := r["employee_name"].(string)
		position, _ := r["job_title_instrumental"].(string)
		department, _ := r["department_genitive"].(string)
		if person == "" || position == "" || department == "" {
			return nil, fmt.Errorf("career: row %d: пустой снапшот имени (сотрудник/должность/отдел)", id)
		}
		key := fmt.Sprintf("%d|%d|%d", emp, dep, job)
		switch ev {
		case "open":
			open[key] = append(open[key], tenure{person: person, position: position, department: department, from: d})
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
