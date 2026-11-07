package main

import (
	"ms_headcount_report/model"
	"ms_headcount_report/print"
	"shared/app"
)

func main() {
	var a app.App[model.Object]
	a.CreateConnection()
	a.Repository.ReportSlots = print.Slots
	a.CreateRoutes()
}
