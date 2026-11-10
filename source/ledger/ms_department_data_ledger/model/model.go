package model

import "time"

// Department data ledger — таймлайн названий отделов (одна строка =
// регистрация или переименование): «название на дату T» = последнее
// с ddate ≤ T. Падежи — все 6 (отчёты склоняют, см. career).
type Object struct {
	Id                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Balance           int64     `gorm:"column:balance" json:"balance"`
	IdDocument        int64     `gorm:"column:id_document" json:"id_document"`
	DDate             time.Time `gorm:"column:ddate" json:"ddate"`
	Prefix            int16     `gorm:"column:prefix" json:"prefix"`
	Name              string    `gorm:"column:name" json:"name"`
	NameGenitive      string    `gorm:"column:name_genitive" json:"name_genitive"`
	NameDative        string    `gorm:"column:name_dative" json:"name_dative"`
	NameAccusative    string    `gorm:"column:name_accusative" json:"name_accusative"`
	NameInstrumental  string    `gorm:"column:name_instrumental" json:"name_instrumental"`
	NamePrepositional string    `gorm:"column:name_prepositional" json:"name_prepositional"`
	IdDepartment      int64     `gorm:"column:id_department" json:"id_department"`
	IdTransaction     string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
