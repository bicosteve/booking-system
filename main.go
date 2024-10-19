package main

import (
	"sync"

	"github.com/bicosteve/booking-system/pkg/controllers"
)

func main() {

	var wg sync.WaitGroup
	var base controllers.Base

	base.Init()

	wg.Add(1)
	go base.AuthAPI(&wg)

	wg.Wait()

}
