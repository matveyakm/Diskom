package model

// Directory of organizations
type Object struct {
	Id            int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DNumber       string `gorm:"column:dnumber" json:"dnumber"`
	Name          string `gorm:"column:name" json:"name"`
	Description   string `gorm:"column:description" json:"description"`
	IdBasis       int64  `gorm:"column:id_basis" json:"id_basis"`
	Status        int16  `gorm:"column:status" json:"status"`
	Prefix        int16  `gorm:"column:prefix" json:"prefix"`
	IdTransaction string `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
