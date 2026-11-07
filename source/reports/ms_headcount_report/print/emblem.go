package print

import (
	"fmt"
	"shared/report"
)

// emblemSlot — логотип в шапку отчёта: символика Дисциплинарного комитета,
// актуальная на дату отчёта (rec ddate — дата проведения). Комитет —
// статическим параметром дырки ("params": {"id_committee": "1"}): отчёт
// глобальный, своего комитета у него нет; эмблема — ДК (id 1).
// Учитываются только проведённые версии (status=2): удалённый приказ
// актуальным быть не может (знание предметной области — здесь, в filler-е).
// Дальше — цепочка версия → файл → байты (см. ActualFileBytes).
// Ноль/несколько файлов у версии — громкая ошибка.
func emblemSlot(ctx report.Ctx, p report.Params) (any, error) {
	org := p["id_committee"]
	if org == "" {
		return nil, fmt.Errorf("emblem: template must set params.id_committee")
	}
	return report.ActualFileBytes(ctx, "symbolism_document", "effective_date",
		report.Str(ctx.Record()["ddate"]),
		map[string]string{
			"id_committee": org,
			"status":       "2",
		})
}
