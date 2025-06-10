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

	wg.Add(5)
	go base.AdminServer(&wg, "7002", "admin")
	go base.UserServer(&wg, "7001", "user")
	go base.RabbitMQConsumer(&wg)

	// if base.RabbitMQStatus == 1 {
	// 	go base.RabbitConsumer(&wg)
	// }

	if base.KafkaStatus == 1 {
		go base.Consumer(&wg, base.Topic[0])
		go base.Consumer(&wg, base.Topic[1])
	}

	defer base.DB.Close()

	wg.Wait()

}
