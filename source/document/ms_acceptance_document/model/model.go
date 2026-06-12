package model

import "time"

// Employee acceptance document
type Object struct {
	Id               int64     `gorm:"column:id;primaryKey" json:"id"`
	Ddate            time.Time `gorm:"column:ddate" json:"ddate"`
	Dnumber          string    `gorm:"column:dnumber" json:"dnumber"`
	Note             string    `gorm:"column:note" json:"note"`
	IdBasis          int16     `gorm:"column:id_basis" json:"id_basis"`
	Status           int16     `gorm:"column:status" json:"status"`
	IdUser           int64     `gorm:"column:id_user" json:"id_user"`
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
	VkId             int64     `gorm:"column:vk_id" json:"vk_id"`
	TransactionId    string    `gorm:"column:id_transaction" json:"transaction_id"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
