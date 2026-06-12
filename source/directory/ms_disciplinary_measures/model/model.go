package model


// Disciplinary Measures model
type Object struct {
	Id		int64	`gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DNumber		string	`gorm:"column:dnumber" json:"dnumber"`
	Name		string	`gorm:"column:name" json:"name"`
	Description		string	`gorm:"column:description" json:"description"`
	Status		int16	`gorm:"column:status" json:"status"`
	IdBasis		int64	`gorm:"column:id_basis" json:"id_basis"`
	Prefix		int16	`gorm:"column:prefix" json:"prefix"`
	IdTransaction		string	`gorm:"column:id_transaction" json:"id_transaction"`
	IdUser		int64	`gorm:"column:id_user" json:"id_user"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
