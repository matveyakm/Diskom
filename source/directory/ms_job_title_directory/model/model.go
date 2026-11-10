package model

// Job title directory
type Object struct {
	Id                int64  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name              string `gorm:"column:name" json:"name"`
	NameGenitive      string `gorm:"column:name_genitive" json:"name_genitive"`
	NameDative        string `gorm:"column:name_dative" json:"name_dative"`
	NameAccusative    string `gorm:"column:name_accusative" json:"name_accusative"`
	NameInstrumental  string `gorm:"column:name_instrumental" json:"name_instrumental"`
	NamePrepositional string `gorm:"column:name_prepositional" json:"name_prepositional"`
	Description       string `gorm:"column:description" json:"description"`
	IdBasis           int64  `gorm:"column:id_basis" json:"id_basis"`
	EndpointBasis     string `gorm:"column:endpoint_basis" json:"endpoint_basis"`
	Status            int16  `gorm:"column:status" json:"status"`
	Prefix            int16  `gorm:"column:prefix" json:"prefix"`
	IdTransaction     string `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
