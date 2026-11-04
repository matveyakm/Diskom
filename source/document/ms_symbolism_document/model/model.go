package model

import "time"

// Symbolism order document (Приказ об установлении символики)
type Object struct {
	Id               int64     `gorm:"column:id;primaryKey" json:"id"`
	DDate            time.Time `gorm:"column:ddate" json:"ddate"`
	DNumber          string    `gorm:"column:dnumber" json:"dnumber"`
	IdCommittee      int64     `gorm:"column:id_committee" json:"id_committee"`
	IdLogo           int64     `gorm:"column:id_logo" json:"id_logo"`
	Comment          string    `gorm:"column:comment" json:"comment"`
	IdBasis          int16     `gorm:"column:id_basis" json:"id_basis"`
	Status           int16     `gorm:"column:status" json:"status"`
	Prefix           int16     `gorm:"column:prefix" json:"prefix"`
	IdUser           int64     `gorm:"column:id_user" json:"id_user"`
	IdTransaction    string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
