package main

import (
	"ms_symbolism_data_ledger/model"
	"shared/app"
)

func main() {
	var a app.App[model.Object]
	a.CreateConnection()
	a.CreateRoutes()
}
