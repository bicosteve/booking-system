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
	"github.com/streadway/amqp"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Base struct {
	KafkaProducer  *kafka.Producer
	KafkaConsumer  *kafka.Consumer
	AuthPort       string
	AdminPort      string
	ConsumerPort   string
	Broker         string
	Topics         []string
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
	paymentService *service.PaymentService
	stripesecret   string
	pubkey         string
	successURL     string
	cancelURL      string
	rabbitConn     *amqp.Connection
	queueName      string
	ctx            context.Context
	KafkaStatus    int
	RabbitMQStatus int
}

func (b *Base) Init() {
	startTime := time.Now()
	ctx := context.Background()
	var brokerURL string
	var paymentKey string
	var paymentTopic []string
	var port int
	var adminport int
	var mqHost string
	var mqPassword string
	var mqPort string
	var mqUser string
	var mqVhost string
	var config entities.Config

	if os.Getenv("ENV") == "prod" {
		conf, err := app.LoadConfigs("env.toml")
		if err != nil {
			os.Exit(1)
		}

		config = conf

	} else {

		conf, err := app.LoadConfigs("env.dev.toml")
		if err != nil {
			os.Exit(1)
		}

		config = conf

	}

	err := utils.InitLogger(config.Logger.Folder)
	if err != nil {
		utils.LogError(err.Error(), entities.ErrorLog)
		os.Exit(1)
	}

	for _, kafka := range config.Kafka {
		brokerURL = kafka.Broker
		paymentKey = kafka.Key
		paymentTopic = kafka.Topics
		b.KafkaStatus = kafka.On

	}

	if b.KafkaStatus == 1 {
		p, err := utils.ProducerConnect(brokerURL)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
			os.Exit(1)
		}

		c, err := utils.ConsumerConnect(brokerURL)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
			os.Exit(1)
		}

		b.KafkaProducer = p
		b.KafkaConsumer = c
		b.Broker = brokerURL
		b.Topics = paymentTopic
		b.Key = paymentKey
	}

	for _, rabbitConf := range config.Rabbit {
		mqHost = rabbitConf.Host
		mqPassword = rabbitConf.Password
		mqPort = rabbitConf.Port
		mqVhost = rabbitConf.Vhost
		b.queueName = rabbitConf.Queue
		mqUser = rabbitConf.User
		b.RabbitMQStatus = rabbitConf.On

	}

	if b.RabbitMQStatus == 1 && os.Getenv("ENV") == "prod" {
		url := fmt.Sprintf("amqp://%s:%s@%s:%s/%s", mqUser, mqPassword, mqHost, mqPort, mqVhost)
		conn, err := utils.NewRabbitMQConnection(url)
		if err != nil {
			os.Exit(1)
		}

		b.rabbitConn = conn
	} else {
		url := fmt.Sprintf("amqp://%s:%s@%s:%s", mqUser, mqPassword, mqHost, mqPort)
		conn, err := utils.NewRabbitMQConnection(url)
		if err != nil {
			os.Exit(1)
		}

		b.rabbitConn = conn
	}

	for _, sql := range config.Mysql {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=latin1&parseTime=True&loc=Local", sql.Username, sql.Password, sql.Host, sql.Port, sql.Schema)
		db, err := connections.DatabaseConnection(dsn)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
			os.Exit(1)
		}

		b.DB = db

	}

	for _, cache := range config.Redis {
		redisClient, err := connections.NewRedisDB(ctx, cache)
		if err != nil {
			utils.LogError(err.Error(), entities.ErrorLog)
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

	for _, secret := range config.Secrets {
		b.jwtSecret = secret.JWT
		b.sengridkey = secret.Sendgrid
		b.mailfrom = secret.MailFrom
		b.atklng = secret.AfricasTalking
		b.appusername = secret.AppUsername
	}

	for _, _stripe := range config.Stripe {
		b.successURL = _stripe.SuccessURL
		b.cancelURL = _stripe.CancelURL
		b.pubkey = _stripe.PubKey
		b.stripesecret = _stripe.StripeSecret
	}

	b.AuthPort = strconv.Itoa(port)
	b.AdminPort = strconv.Itoa(adminport)

	// Initializing user repo
	userRepository := repo.NewDBRepository(b.DB, b.Redis)
	userService := service.NewUserService(*userRepository)
	b.userService = userService

	// Initializing room repo
	roomRepository := repo.NewDBRepository(b.DB, b.Redis)
	roomService := service.NewRoomService(*roomRepository)
	b.roomService = roomService

	// Initialize booking repo
	bookingRepository := repo.NewDBRepository(b.DB, b.Redis)
	bookingService := service.NewBookingService(*bookingRepository)
	b.bookingService = bookingService

	// Initialize payment repo
	paymentRepository := repo.NewDBRepository(b.DB, b.Redis)
	paymentService := service.NewPaymentService(*paymentRepository)
	b.paymentService = paymentService

	_msg := fmt.Sprintf("Connections done in %v\n", time.Since(startTime))
	utils.LogInfo(_msg, entities.InfoLog)

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
		utils.LogError(err.Error(), entities.ErrorLog)
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
		utils.LogError(err.Error(), entities.ErrorLog)
		os.Exit(1)
	}

}

func (b *Base) userRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	utils.SetCors(r)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:7001/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
	))

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
		r.Get("/user/book/verify/{room_id}", b.VerifyBookingHandler)
		r.Get("/user/book/{room_id}", b.GetBookingHandler)
		r.Get("/user/book/all", b.GetAllBookingsHandler)
		r.Put("/user/book/{booking_id}", b.UpdateBooking)

	})

	return r

}

func (b *Base) adminRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	utils.SetCors(router)

	router.Mount("/swagger", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:7002/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
	))

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
