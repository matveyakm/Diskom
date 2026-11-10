package model

import "time"

// Staff placement ledger: append-only journal of position open/close events.
// One row per event; the career report pairs open/close rows FIFO per
// (employee, department, job title) triple. Display names are resolved
// at report time from registers (employee_data fio timeline +
// *_data_ledger name timelines as of the report date) — no snapshots here:
// snapshots froze event-time names and contradicted formation-date display.
type Object struct {
	Id             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Endpoint       string    `gorm:"column:endpoint" json:"endpoint"`
	Event          string    `gorm:"column:event" json:"event"`
	IdDocument     int64     `gorm:"column:id_document" json:"id_document"`
	DDate          time.Time `gorm:"column:ddate;index:idx_placement_ddate" json:"ddate"`
	Prefix         int16     `gorm:"column:prefix" json:"prefix"`
	IdEmployee     int64     `gorm:"column:id_employee;index:idx_placement_employee" json:"id_employee"`
	IdOrganization int64     `gorm:"column:id_organization" json:"id_organization"`
	IdDepartment   int64     `gorm:"column:id_department" json:"id_department"`
	IdJobTitle     int64     `gorm:"column:id_job_title" json:"id_job_title"`
	Justification  string    `gorm:"column:justification" json:"justification"`
	IdTransaction  string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
