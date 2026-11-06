// Package print — индивидуальный код печати приёма: filler-ы слотов
// шаблона print/print.json + их реестр. Главный движок (shared) документ
// не смотрит: связь дырки и функции — совпадением имён.
package print

import (
	"shared/report"
)

// emblemSlot — логотип в шапку приказа: символика комитета приказа,
// актуальная на его дату. Комитет — полем строки (id_committee), дата —
// ddate. Учитываются только проведённые версии (status=2): удалённый приказ
// актуальным быть не может (это знание предметной области — здесь, в filler-е,
// а не в generic actual). Дальше — известная цепочка: версия → файл → байты.
// Ноль/несколько файлов у версии — громкая ошибка (см. ActualFileBytes).
func emblemSlot(ctx report.Ctx, _ report.Params) (any, error) {
	return report.ActualFileBytes(ctx, "symbolism_document", "effective_date",
		report.Str(ctx.Record()["ddate"]),
		map[string]string{
			"id_committee": report.Str(ctx.Record()["id_committee"]),
			"status":       "2",
		})
}

// Slots — реестр дырок печати сервиса (shared ничего про документ
// не знает: связь — совпадением имён, смысл — в позиции в шаблоне).
var Slots = report.Slots{"emblem": emblemSlot}
