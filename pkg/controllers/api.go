package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

func (b *Base) AuthAPI(wg *sync.WaitGroup) {
	defer wg.Done()

	port := ":7001"

	srv := &http.Server{
		Addr:    port,
		Handler: b.Router(),
	}

	fmt.Printf("Listening to port %s \n", port)
	err := srv.ListenAndServe()
	if err != nil {
		log.Printf("Error running auth %s \n", err)
		os.Exit(1)
	}
}
