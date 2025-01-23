package controllers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bicosteve/booking-system/connections"
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/app"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/bicosteve/booking-system/repo"
	"github.com/bicosteve/booking-system/service"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
)

type Base struct {
	KafkaProducer *kafka.Producer
	KafkaConsumer *kafka.Consumer
	AuthPort      string
	AdminPort     string
	ConsumerPort  string
	Broker        string
	Topic         string
	Key           string
	DB            *sql.DB
	Redis         *redis.Conn
	ctx           context.Context
	jwtSecret     string
	contentType   string
	path          string
}

func (b *Base) Init() {
	startTime := time.Now()
	ctx := context.Background()
	var brokerURL string
	var authKey string
	var authTopic string
	var port int
	var adminport int

	config, err := app.LoadConfigs("booking_system.toml")
	if err != nil {
		os.Exit(1)
	}

	for _, kafka := range config.Kafka {
		brokerURL = kafka.Broker
		authKey = kafka.Key
		authTopic = kafka.Topic

	}

	p, err := utils.ProducerConnect(brokerURL)
	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("Error connecting because of %s\n", err)
		os.Exit(1)
	}

	c, err := utils.ConsumerConnect(brokerURL)
	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("Error connecting because of %s\n", err)
		os.Exit(1)
	}

	for _, sql := range config.Mysql {
		mysqldb, err := connections.ConnectSQLDB(ctx, sql)
		if err != nil {
			entities.MessageLogs.ErrorLog.Printf("BASE: Could not connect db due to %v", err)
			os.Exit(1)
		}

		b.DB = mysqldb.Connection

	}

	for _, cache := range config.Redis {
		redis, err := connections.NewRedisDB(ctx, cache)
		if err != nil {
			entities.MessageLogs.ErrorLog.Printf("BASE: Could not connect redis due to %v", err)
			os.Exit(1)
		}

		b.Redis = redis.Client.Conn()

	}

	for _, p := range config.Http {
		port = p.Port
		adminport = p.AdminPort
		b.contentType = p.ContentType
		b.path = p.Path

	}

	for _, s := range config.Secrets {
		b.jwtSecret = s.JWT
	}

	b.AuthPort = strconv.Itoa(port)
	b.AdminPort = strconv.Itoa(adminport)
	b.KafkaProducer = p
	b.KafkaConsumer = c
	b.Broker = brokerURL
	b.Topic = authTopic
	b.Key = authKey

	entities.MessageLogs.InfoLog.Printf("Connections done in %v\n", time.Since(startTime))

}

func (b *Base) UserServer(wg *sync.WaitGroup, port, server string) {
	defer wg.Done()

	userSRV := &http.Server{
		Addr:    ":" + port,
		Handler: b.userRouter(),
	}

	fmt.Printf("Listening to %v server on port %s \n", server, port)
	err := userSRV.ListenAndServe()
	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("error running user server %v", err)
		os.Exit(1)
	}

}

func (b *Base) AdminServer(wg *sync.WaitGroup, port, server string) {
	defer wg.Done()

	userSRV := &http.Server{
		Addr:    ":" + port,
		Handler: b.adminRouter(),
	}

	fmt.Printf("Listening to %v server on port %s \n", server, port)
	err := userSRV.ListenAndServe()
	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("error running user server %v", err)
		os.Exit(1)
	}

}

func (b *Base) userRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	utils.SetCors(r)

	repo := repo.NewUserDBRepository(b.DB, b.ctx)
	service := service.NewUserService(*repo)

	// Public Routes
	r.Post(b.path+"/user/register", b.RegisterHandler(service))
	r.Post(b.path+"/user/login", b.LoginHandler(service))

	// Private routes
	r.Route(b.path, func(r chi.Router) {
		r.Use(utils.AuthMiddleware(b.jwtSecret))
		r.Get("/user/me", b.ProfileHandler(service))
		r.Post("/user/reset", b.GenerateResetTokenHandler(service))

	})

	return r

}

func (b *Base) adminRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	utils.SetCors(router)

	return router

}
