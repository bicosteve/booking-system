package controllers

import "sync"

func AuthConsumer(wg *sync.WaitGroup) {
	defer wg.Done()

}
