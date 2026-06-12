package model

import "time"

// Employee acceptance document
type Object struct {
	Id             int64     `gorm:"column:id;primaryKey" json:"id"`
	Ddate          time.Time `gorm:"column:ddate" json:"ddate"`
	Dnumber        string    `gorm:"column:dnumber" json:"dnumber"`
	Name           string    `gorm:"column:name" json:"name"`
	Note           string    `gorm:"column:note" json:"note"`
	IdBasis        int64     `gorm:"column:id_basis" json:"id_basis"`
	Status         int16     `gorm:"column:status" json:"status"`
	IdUser         int64     `gorm:"column:id_user" json:"id_user"`
	Prefix         int16     `gorm:"column:prefix" json:"prefix"`
	IdOrganization int64     `gorm:"column:id_organization" json:"id_organization"`
	TransactionId  string    `gorm:"column:id_transaction" json:"transaction_id"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
