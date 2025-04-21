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
	KafkaProducer  *kafka.Producer
	KafkaConsumer  *kafka.Consumer
	AuthPort       string
	AdminPort      string
	ConsumerPort   string
	Broker         string
	Topic          string
	Key            string
	DB             *sql.DB
	Redis          *redis.Client
	jwtSecret      string
	contentType    string
	path           string
	sengridkey     string
	mailfrom       string
	atklng         string
	appusername    string
	userService    *service.UserService
	roomService    *service.RoomService
	bookingService *service.BookingService
	pp_clientid    string
	pp_secret      string
	stripesecret   string
}

func (b *Base) Init() {
	startTime := time.Now()
	ctx := context.Background()
	var brokerURL string
	// var authKey string
	// var authTopic string
	var paymentKey string
	var paymentTopic string
	var port int
	var adminport int

	config, err := app.LoadConfigs("booking_system.toml")
	if err != nil {
		os.Exit(1)
	}

	for _, kafka := range config.Kafka {
		brokerURL = kafka.Broker
		// authKey = kafka.Key
		// authTopic = kafka.Topic
		paymentKey = kafka.Key
		paymentTopic = kafka.Topic

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
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=latin1&parseTime=True&loc=Local", sql.Username, sql.Password, sql.Host, sql.Port, sql.Schema)
		db, err := connections.DatabaseConnection(dsn)
		if err != nil {
			entities.MessageLogs.ErrorLog.Printf("BASE: Could not connect db due to %v", err)
			os.Exit(1)
		}

		b.DB = db

	}

	for _, cache := range config.Redis {
		redisClient, err := connections.NewRedisDB(ctx, cache)
		if err != nil {
			entities.MessageLogs.ErrorLog.Printf("BASE: Could not connect redis due to %v", err)
			os.Exit(1)
		}

		b.Redis = redisClient

	}

	for _, p := range config.Http {
		port = p.Port
		adminport = p.AdminPort
		b.contentType = p.ContentType
		b.path = p.Path

	}

	for _, s := range config.Secrets {
		b.jwtSecret = s.JWT
		b.sengridkey = s.Sendgrid
		b.mailfrom = s.MailFrom
		b.atklng = s.AfricasTalking
		b.appusername = s.AppUsername
		b.pp_clientid = s.PPClientID
		b.pp_secret = s.PPSecret
		b.stripesecret = s.StripeSecret
	}

	b.AuthPort = strconv.Itoa(port)
	b.AdminPort = strconv.Itoa(adminport)
	b.KafkaProducer = p
	b.KafkaConsumer = c
	b.Broker = brokerURL
	b.Topic = paymentTopic
	b.Key = paymentKey

	// Initializing user repo
	userRepository := repo.NewDBRepository(b.DB)
	userService := service.NewUserService(*userRepository)
	b.userService = userService

	// Initializing room repo
	roomRepository := repo.NewDBRepository(b.DB)
	roomService := service.NewRoomService(*roomRepository)
	b.roomService = roomService

	// Initialize booking repo
	bookingRepository := repo.NewDBRepository(b.DB)
	bookingService := service.NewBookingService(*bookingRepository)
	b.bookingService = bookingService

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

	// Public Routes
	r.Post(b.path+"/user/register", b.RegisterHandler)
	r.Post(b.path+"/user/login", b.LoginHandler)
	r.Get(b.path+"/user/rooms", b.FindRoomHandler)

	// Private routes
	r.Route(b.path, func(r chi.Router) {
		r.Use(utils.AuthMiddleware(b.jwtSecret))
		r.Get("/user/me", b.ProfileHandler)
		r.Post("/user/reset", b.GenerateResetTokenHandler)
		r.Post("/user/password-reset", b.ResetPasswordHandler)
		r.Post("/user/book", b.CreateBookingHandler)
		r.Get("/user/book/{booking_id}", b.GetBookingHandler)
		r.Get("/user/book/all", b.GetAllBookingsHandler)
		r.Put("/user/book/{booking_id}", b.UpdateBooking)

	})

	return r

}

func (b *Base) adminRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	utils.SetCors(router)

	router.Route(b.path, func(r chi.Router) {
		r.Use(utils.AuthMiddleware(b.jwtSecret))
		r.Use(utils.AdminMiddlware)
		r.Post("/admin/rooms", b.CreateRoomHandler)
		r.Put("/admin/rooms/{room_id}", b.UpdateARoom)
		r.Delete("/admin/rooms/{room_id}", b.DeleteARoom)
		r.Get("/admin/book/all", b.GetAllAdminBookingsHandler)
		r.Delete("/admin/book/{booking_id}/{room_id}", b.DeleteBooking)

	})

	return router

}
