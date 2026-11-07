package model

import "time"

// Employee data ledger — остатки штата (одна строка = изменение).
// balance — накопленный итог, считает сервер (ACCUMULATE=balance:delta:ddate):
// запрос присылает только delta (+1/-1), явный balance — 400.
// endpoint — тип породившего документа (литерал из конфига пайплайна).
type Object struct {
	Id         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DDate      time.Time `gorm:"column:ddate" json:"ddate"`
	Endpoint   string    `gorm:"column:endpoint" json:"endpoint"`
	IdDocument int64     `gorm:"column:id_document" json:"id_document"`
	IdEmployee int64     `gorm:"column:id_employee" json:"id_employee"`
	Delta      int64     `gorm:"column:delta" json:"delta"`
	Balance    int64     `gorm:"column:balance" json:"balance"`
}

func (o *Object) GetID() int64 {
	return o.Id
}
