package main

import (
	"ms_acceptance_document/model"
	"ms_acceptance_document/print"
	"shared/app"
)

func main() {
	var a app.App[model.Object]
	a.CreateConnection()
	a.Repository.ReportSlots = print.Slots
	a.CreateRoutes()
}
