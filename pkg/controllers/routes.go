package controllers

import (
	"net/http"

	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (b *Base) Router() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	utils.SetCors(router)

	router.Get("/ping", b.Register)

	return router

}
