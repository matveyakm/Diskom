package model

import "time"

// Symbolism data ledger. Файлов в проводке нет: файлы живут в files
// (связь по id_document), строка документа файловых колонок не имеет.
type Object struct {
	Id               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	IdDocument       int64     `gorm:"column:id_document" json:"id_document"`
	DDate            time.Time `gorm:"column:ddate" json:"ddate"`
	EffectiveDate    time.Time `gorm:"column:effective_date" json:"effective_date"`
	IdCommittee      int64     `gorm:"column:id_committee" json:"id_committee"`
	Comment          string    `gorm:"column:comment" json:"comment"`
	Prefix           int16     `gorm:"column:prefix" json:"prefix"`
	IdTransaction    string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
