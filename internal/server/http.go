package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bicosteve/booking-system/pkg/entities"
)

type httpserver struct {
	server *http.Server
	ctx    context.Context
	config entities.HttpConfig
}

type HttpMessage struct {
	StatusCode  string `json:"status_code"`
	Data        any    `json:"data"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
}

func NewHttpServer(ctx context.Context, config entities.HttpConfig, handler http.Handler) error {
	slog.Info("Http server starting ...")
	slog.Debug(fmt.Sprintf("Http: host: %v | Port: %v | Path: %v ", config.Host, config.Port, config.Path))
	slog.Debug(fmt.Sprintf("Http: Cors: %v", config.Cors))
	slog.Debug(fmt.Sprintf("Http: Args: %v", config.Args))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%v", config.Port),
		Handler: handler,
	}

	hs := &httpserver{
		ctx:    ctx,
		server: server,
		config: config,
	}

	go func() {
		serverError := hs.server.ListenAndServe()
		if serverError != http.ErrServerClosed {
			slog.Error(fmt.Errorf("http server [:%v] failed to start : %v", hs.config.Port, serverError).Error())
			return
		}
	}()

	slog.Info(fmt.Sprintf("Http server {%v} started ...", hs.server.Addr))
	go hs.close()

	return nil
}

func (hs *httpserver) close() error {
	<-hs.ctx.Done()
	return hs.server.Shutdown(hs.ctx)
}

func NewHttpMessage(statusCode string, data any, description string) HttpMessage {
	return HttpMessage{
		StatusCode:  statusCode,
		Data:        data,
		Description: description,
		Timestamp:   time.Now().Format("2006-01-02 15:04:05.000"),
	}
}
