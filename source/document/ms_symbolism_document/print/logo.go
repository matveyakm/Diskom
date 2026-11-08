package print

import (
	"fmt"
	"shared/report"
)

// logoSlot — изображение приказа: файл, приложенный к самому документу
// (связь 1:N — только files.id_document, без колонок и ролей в строке).
// Строго ровно 1 файл: ноль — недописанный приказ, несколько —
// неопределённость, что утверждаем (неверный приказ хуже отсутствия).
// Черновики сюда не доходят: слоты ленивые, в невзятой ветке if filler
// не вызывается (там распор).
func logoSlot(ctx report.Ctx, _ report.Params) (any, error) {
	id, ok := report.Int(ctx.Record()["id"])
	if !ok {
		return nil, fmt.Errorf("logo: no id in record")
	}
	files, err := ctx.FilesOf("symbolism_document", id)
	if err != nil {
		return nil, fmt.Errorf("logo: files of #%d: %w", id, err)
	}
	if len(files) != 1 {
		return nil, fmt.Errorf("logo: want exactly 1 attached file, got %d", len(files))
	}
	data, mime, err := ctx.FileBytes("symbolism_document", files[0].ID, id)
	if err != nil {
		return nil, fmt.Errorf("logo: file #%d: %w", files[0].ID, err)
	}
	if mime != "image/png" && mime != "image/jpeg" {
		return nil, fmt.Errorf("logo: file #%d is %s, want png/jpg image", files[0].ID, mime)
	}
	return data, nil
}

// Slots — реестр дырок печати сервиса (shared ничего про документ
// не знает: связь — совпадением имён, смысл — в позиции в шаблоне).
var Slots = report.Slots{"logo": logoSlot}
