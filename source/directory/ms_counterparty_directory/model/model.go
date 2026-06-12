package model

type Object struct {
	Id            int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DNumber       string `gorm:"column:dnumber" json:"dnumber"`
	Name          string `gorm:"column:name;not null" json:"name"`
	Description   string `gorm:"column:description" json:"description"`
	IdBasis       int64  `gorm:"column:id_basis" json:"id_basis"`
	Status        int16  `gorm:"column:status;default:1" json:"status"`
	Prefix        int16  `gorm:"column:prefix" json:"prefix"`
	FirstName     string `gorm:"column:first_name" json:"first_name"`
	LastName      string `gorm:"column:last_name" json:"last_name"`
	MiddleName    string `gorm:"column:middle_name" json:"middle_name"`
	IdTransaction string `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
