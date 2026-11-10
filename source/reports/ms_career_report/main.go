package main

import (
	"ms_career_report/model"
	"ms_career_report/print"
	"shared/app"
)

func main() {
	var a app.App[model.Object]
	a.CreateConnection()
	a.Repository.ReportSlots = print.Slots
	a.CreateRoutes()
}
