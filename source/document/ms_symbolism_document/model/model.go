package model

import "time"

// Symbolism order document (Приказ об установлении символики).
// Файлов в строке НЕТ (связь 1:N — только files.id_document; файловое поле
// формы виртуально: живёт в контексте пайплайна, в строку не пишется).
type Object struct {
	Id            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DDate         time.Time `gorm:"column:ddate" json:"ddate"`
	EffectiveDate time.Time `gorm:"column:effective_date" json:"effective_date"`
	DNumber       string    `gorm:"column:dnumber" json:"dnumber"`
	IdCommittee   int64     `gorm:"column:id_committee" json:"id_committee"`
	Comment       string    `gorm:"column:comment" json:"comment"`
	IdBasis       int64     `gorm:"column:id_basis" json:"id_basis"`
	EndpointBasis string    `gorm:"column:endpoint_basis" json:"endpoint_basis"`
	Status        int16     `gorm:"column:status" json:"status"`
	Prefix        int16     `gorm:"column:prefix" json:"prefix"`
	IdUser        int64     `gorm:"column:id_user" json:"id_user"`
	IdTransaction string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
