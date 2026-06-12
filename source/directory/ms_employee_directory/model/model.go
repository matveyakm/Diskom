package model

// Employee directory
type Object struct {
	Id            int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DNumber       string `gorm:"column:dnumber" json:"dnumber"`
	Name          string `gorm:"column:name;not null" json:"name"`
	Description   string `gorm:"column:description" json:"description"`
	IdBasis       int16  `gorm:"column:id_basis" json:"id_basis"`
	Status        int16  `gorm:"column:status;default:1" json:"status"`
	VkId          int64  `gorm:"column:vk_id" json:"vk_id"`
	IdTransaction string `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
