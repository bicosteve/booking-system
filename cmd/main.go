package main

import (
	"sync"

	"github.com/bicosteve/booking-system/controllers"
	_ "github.com/bicosteve/booking-system/docs"
)

// @title Booking API
// @version 1.0
// @Description Booking API to make booking reservations

// @contact.name Bico Oloo
// @contact.url https://github.com/bicosteve
// @contact.email bicosteve4@gmail.com

// @BasePath /api
// @schemes http

func main() {

	var wg sync.WaitGroup
	var base controllers.Base

	base.Init()

	wg.Add(3)
	go base.AdminServer(&wg, "7002", "admin")
	go base.UserServer(&wg, "7001", "user")
	go base.RabbitMQConsumer(&wg)

	// go base.Consumer(&wg, base.Topics[0])
	// go base.Consumer(&wg, base.Topics[1])

	defer base.DB.Close()

	wg.Wait()

}
