package model

import "time"

// Employee dismissal document
type Object struct {
	Id             int64     `gorm:"column:id;primaryKey" json:"id"`
	Ddate          time.Time `gorm:"column:ddate" json:"ddate"`
	Dnumber        string    `gorm:"column:dnumber" json:"dnumber"`
	Note           string    `gorm:"column:note" json:"note"`
	IdBasis        int64     `gorm:"column:id_basis" json:"id_basis"`
	Status         int16     `gorm:"column:status" json:"status"`
	IdUser         int64     `gorm:"column:id_user" json:"id_user"`
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
