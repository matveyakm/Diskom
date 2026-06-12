package model

import "time"

// Employee data ledger
type Object struct {
	Id               int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Balance          int64     `gorm:"column:balance" json:"balance"`
	IdDocument       int64     `gorm:"column:id_document" json:"id_document"`
	DDate            time.Time `gorm:"column:ddate" json:"ddate"`
	Prefix           int16     `gorm:"column:prefix" json:"prefix"`
	IdEmployee       int64     `gorm:"column:id_employee" json:"id_employee"`
	FirstName        string    `gorm:"column:first_name" json:"first_name"`
	LastName         string    `gorm:"column:last_name" json:"last_name"`
	MiddleName       string    `gorm:"column:middle_name" json:"middle_name"`
	FirstGenitive    string    `gorm:"column:first_genitive" json:"first_genitive"`
	LastGenitive     string    `gorm:"column:last_genitive" json:"last_genitive"`
	MiddleGenitive   string    `gorm:"column:middle_genitive" json:"middle_genitive"`
	FirstAccusative  string    `gorm:"column:first_accusative" json:"first_accusative"`
	LastAccusative   string    `gorm:"column:last_accusative" json:"last_accusative"`
	MiddleAccusative string    `gorm:"column:middle_accusative" json:"middle_accusative"`
	IdVk             int64     `gorm:"column:id_vk" json:"id_vk"`
	IdTransaction    string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
