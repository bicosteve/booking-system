package main

import (
	"sync"

	"github.com/bicosteve/booking-system/controllers"
)

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
