package model

import "time"

// Headcount report (Отчёт о численности сотрудников за период).
// Сущность-документ: строка = один запуск отчёта с параметрами
// (период; учёт глобальный, без разреза по комитету — регистр копит общий
// итог). Проводок не пишет (acceptance только меняет статус).
type Object struct {
	Id            int64     `gorm:"column:id;primaryKey" json:"id"`
	DDate         time.Time `gorm:"column:ddate" json:"ddate"`
	DNumber       string    `gorm:"column:dnumber" json:"dnumber"`
	DateFrom      time.Time `gorm:"column:date_from" json:"date_from"`
	DateTo        time.Time `gorm:"column:date_to" json:"date_to"`
	Comment       string    `gorm:"column:comment" json:"comment"`
	IdBasis       int16     `gorm:"column:id_basis" json:"id_basis"`
	Status        int16     `gorm:"column:status" json:"status"`
	Prefix        int16     `gorm:"column:prefix" json:"prefix"`
	IdUser        int64     `gorm:"column:id_user" json:"id_user"`
	IdTransaction string    `gorm:"column:id_transaction" json:"id_transaction"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
