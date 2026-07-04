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
	"github.com/bicosteve/booking-system/pkg/health"
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
	rabbitURL      string
	rabbitCfg      entities.RabbitMQConfig
	kafkaCfg       entities.KakfaConfig
	// checkersProvider is overridden in tests; nil means use defaultLiveCheckers(). Used by HealthCheck.
	checkersProvider func() []health.Checker
	ctx              context.Context
	KafkaStatus      int
	RabbitMQStatus   int
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

		kafkaStatus, _ := strconv.Atoi(os.Getenv("KAFKA_STATUS"))
		rabbitMQStatus, _ := strconv.Atoi(os.Getenv("RABBITMQ_STATUS"))
		dbPort, _ := strconv.Atoi(os.Getenv("DB_PORT"))
		redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
		userPort, _ := strconv.Atoi(os.Getenv("HTTP_PORT"))
		adminPort, _ := strconv.Atoi(os.Getenv("ADMIN_PORT"))

		config = entities.Config{
			Logger: entities.LoggerConfig{Folder: os.Getenv("LOGGER_FOLDER")},
			Kafka: []entities.KakfaConfig{
				{
					Broker: os.Getenv("KAFKA_BROKER"),
					Key:    os.Getenv("KAFKA_KEY"),
					Topics: []string{os.Getenv("KAFKA_TOPIC")},
					On:     kafkaStatus,
				},
			},
			Rabbit: []entities.RabbitMQConfig{
				{
					Host:     os.Getenv("RABBITMQ_HOST"),
					Port:     os.Getenv("RABBITMQ_PORT"),
					User:     os.Getenv("RABBITMQ_USER"),
					Password: os.Getenv("RABBITMQ_PASSWORD"),
					Vhost:    os.Getenv("RABBITMQ_VHOST"),
					Queue:    os.Getenv("RABBITMQ_QUEUE"),
					On:       rabbitMQStatus,
				},
			},
			Mysql: []entities.MysqlConfig{
				{
					Username: os.Getenv("DB_USER"),
					Password: os.Getenv("DB_PASSWORD"),
					Host:     os.Getenv("DB_HOST"),
					Port:     dbPort,
					Schema:   os.Getenv("DB_SCHEMA"),
				},
			},
			Redis: []entities.RedisConfig{
				{
					Name:     os.Getenv("REDIS_NAME"),
					Address:  os.Getenv("REDIS_ADDRESS"),
					Port:     os.Getenv("REDIS_PORT"),
					Password: os.Getenv("REDIS_PASSWORD"),
					Database: redisDB,
				},
			},
			Http: []entities.HttpConfig{
				{
					Port:        userPort,
					AdminPort:   adminPort,
					ContentType: os.Getenv("CONTENT_TYPE"),
					Path:        os.Getenv("API_PATH"),
				},
			},
			Secrets: []entities.SecretConfig{
				{
					Name:           "secrets",
					JWT:            os.Getenv("JWT_SECRET"),
					Sendgrid:       os.Getenv("SENDGRID_KEY"),
					MailFrom:       os.Getenv("MAIL_FROM"),
					AfricasTalking: os.Getenv("AT_KEY"),
					AppUsername:    os.Getenv("APP_USERNAME"),
					PPClientID:     os.Getenv("PP_CLIENT_ID"),
					PPSecret:       os.Getenv("PP_SECRET"),
					StripeSecret:   os.Getenv("STRIPE_SECRET"),
				},
			},
			Stripe: []entities.StripeConfig{
				{
					Name:         os.Getenv("STRIPE_NAME"),
					StripeSecret: os.Getenv("STRIPE_SECRET"),
					PubKey:       os.Getenv("STRIPE_PUB_KEY"),
					SuccessURL:   os.Getenv("STRIPE_SUCCESS_URL"),
					CancelURL:    os.Getenv("STRIPE_CANCEL_URL"),
				},
			},
		}

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

	// Wait for backing services to be reachable before connecting, so the app
	// doesn't exit when docker-compose services come up at different times.
	b.waitForDependencies(config)

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
		b.rabbitURL = url
		conn, err := utils.NewRabbitMQConnection(url)
		if err != nil {
			os.Exit(1)
		}

		b.rabbitConn = conn
	} else {
		url := fmt.Sprintf("amqp://%s:%s@%s:%s", mqUser, mqPassword, mqHost, mqPort)
		b.rabbitURL = url
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

	// Store the base context so background workers (e.g. the RabbitMQ consumer)
	// have a non-nil context to pass down to the service/repo layers.
	b.ctx = ctx

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

	swaggerURL := ""
	if os.Getenv("ENV") == "prod" {
		swaggerURL = "swagger/doc.json"
	} else {
		swaggerURL = "http://localhost:7001/swagger/doc.json"
	}

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(swaggerURL),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
	))

	// Public Routes
	r.Post(b.path+"/user/register", b.RegisterHandler)
	r.Post(b.path+"/user/login", b.LoginHandler)
	r.Get(b.path+"/user/rooms", b.FindRoomHandler)
	r.Get(b.path+"/health/test", b.HealthCheck)

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
	swaggerURL := ""
	if os.Getenv("ENV") == "prod" {
		swaggerURL = "swagger/doc.json"
	} else {
		swaggerURL = "http://localhost:7001/swagger/doc.json"
	}

	router.Mount("/swagger", httpSwagger.Handler(
		httpSwagger.URL(swaggerURL),
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
