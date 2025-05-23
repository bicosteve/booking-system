package main

import (
	"sync"

	_ "github.com/bicosteve/booking-system/cmd/docs"
	"github.com/bicosteve/booking-system/controllers"
)

// @title Booking API
// @version 1
// @Description Booking API to perform booking CRUD operations

// @contact.name Bico Oloo
// @contact.url https://github.com/bicosteve
// @contact.email bicosteve4@gmail.com

// @host.user localhost:7001
// @host.admin localhost:7002
// @BasePath /api

func main() {

	var wg sync.WaitGroup
	var base controllers.Base

	base.Init()

	wg.Add(4)
	go base.AdminServer(&wg, "7002", "admin")
	go base.UserServer(&wg, "7001", "user")
	go base.Consumer(&wg, base.Topic[0])
	go base.Consumer(&wg, base.Topic[1])

	defer base.DB.Close()

	wg.Wait()

}
