package main

import (
	"ms_dismissal_document/model"
	"shared/app"
)

func main() {
	var a app.App[model.Object]
	a.CreateConnection()
	a.CreateRoutes()
}
