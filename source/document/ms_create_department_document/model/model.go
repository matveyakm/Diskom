package model

import "time"

// Employee acceptance document
type Object struct {
	Id                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Ddate             time.Time `gorm:"column:ddate" json:"ddate"`
	Dnumber           string    `gorm:"column:dnumber" json:"dnumber"`
	Name              string    `gorm:"column:name" json:"name"`
	NameGenitive      string    `gorm:"column:name_genitive" json:"name_genitive"`
	NameDative        string    `gorm:"column:name_dative" json:"name_dative"`
	NameAccusative    string    `gorm:"column:name_accusative" json:"name_accusative"`
	NameInstrumental  string    `gorm:"column:name_instrumental" json:"name_instrumental"`
	NamePrepositional string    `gorm:"column:name_prepositional" json:"name_prepositional"`
	Note              string    `gorm:"column:note" json:"note"`
	IdBasis           int64     `gorm:"column:id_basis" json:"id_basis"`
	EndpointBasis     string    `gorm:"column:endpoint_basis" json:"endpoint_basis"`
	Status            int16     `gorm:"column:status" json:"status"`
	IdUser            int64     `gorm:"column:id_user" json:"id_user"`
	Prefix            int16     `gorm:"column:prefix" json:"prefix"`
	IdDepartment      int64     `gorm:"column:id_department" json:"id_department"`
	TransactionId     string    `gorm:"column:id_transaction" json:"transaction_id"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
