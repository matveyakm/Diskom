package main

import (
	"ms_reasons_for_the_dcs_disagreement/model"
	"shared/app"
)

func main() {
	var a app.App[model.Object]
	a.CreateConnection()
	a.CreateRoutes()
}
