package model

import "time"

// Document
type Object struct {
	Id             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Ddate          time.Time `gorm:"column:ddate" json:"ddate"`
	Dnumber        string    `gorm:"column:dnumber" json:"dnumber"`
	Note           string    `gorm:"column:note" json:"note"`
	IdBasis        int64     `gorm:"column:id_basis" json:"id_basis"`
	Status         int16     `gorm:"column:status" json:"status"`
	IdUser         int64     `gorm:"column:id_user" json:"id_user"`
	Prefix         int16     `gorm:"column:prefix" json:"prefix"`
	IdSystemObject int64     `gorm:"column:id_system_object;" json:"id_system_object"`
	IdTransaction  string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
