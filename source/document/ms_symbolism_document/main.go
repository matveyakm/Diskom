package main

import (
	"ms_symbolism_document/model"
	"ms_symbolism_document/print"
	"shared/app"
)

func main() {
	var a app.App[model.Object]
	a.CreateConnection()
	a.Repository.ReportSlots = print.Slots
	a.CreateRoutes()
}
