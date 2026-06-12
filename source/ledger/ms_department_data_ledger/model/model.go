package model

import "time"

// Employee data ledger
type Object struct {
	Id            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Balance       int64     `gorm:"column:balance" json:"balance"`
	IdDocument    int64     `gorm:"column:id_document" json:"id_document"`
	DDate         time.Time `gorm:"column:ddate" json:"ddate"`
	Prefix        int16     `gorm:"column:prefix" json:"prefix"`
	Name          string    `gorm:"column:name" json:"name"`
	IdDepartment  int64     `gorm:"column:id_department" json:"id_department"`
	IdTransaction string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
