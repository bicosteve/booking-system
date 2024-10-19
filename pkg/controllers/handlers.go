package controllers

import (
	"net/http"
)

func (b *Base) Register(w http.ResponseWriter, r *http.Request) {
	payload := "This is payload"
	w.Write([]byte(payload))
	//fmt.Println(payload)
}
