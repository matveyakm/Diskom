package model

import "time"

// Staff placement ledger
type Object struct {
	Id             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Balance        int16     `gorm:"column:balance" json:"balance"`
	IdDocument     int64     `gorm:"column:id_document" json:"id_document"`
	DDate          time.Time `gorm:"column:ddate" json:"ddate"`
	Prefix         int16     `gorm:"column:prefix" json:"prefix"`
	IdEmployee     int64     `gorm:"column:id_employee" json:"id_employee"`
	IdOrganization int64     `gorm:"column:id_organization" json:"id_organization"`
	IdDepartment   int64     `gorm:"column:id_department" json:"id_department"`
	IdJobTitle     int64     `gorm:"column:id_job_title" json:"id_job_title"`
	Justification  string    `gorm:"column:justification" json:"justification"`
	IdTransaction  string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
