package controllers

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/bicosteve/booking-system/pkg/app"
	"github.com/bicosteve/booking-system/pkg/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/redis/go-redis/v9"
)

type Base struct {
	KafkaProducer *kafka.Producer
	KafkaConsumer *kafka.Consumer
	Cache         *redis.Client
	AuthPort      string
	ConsumerPort  string
	Broker        string
	Topic         string
	Key           string
	DB            *sql.DB
}

func (b *Base) Init() {
	startTime := time.Now()
	// ctx := context.Background()

	// Load configs from toml file
	config, err := app.LoadConfigs("booking_system.toml")
	if err != nil {
		os.Exit(1)
	}

	fmt.Println(config)

	// err = godotenv.Load(".env")
	// if err != nil {
	// 	entities.MessageLogs.ErrorLog.Printf("Error loading .env file %s\n", err)
	// 	os.Exit(1)
	// }

	// brokerURL := os.Getenv("AUTHBROKER")
	// authTopic := os.Getenv("AUTHTOPIC")
	// authKey := os.Getenv("AUTHKEY")
	// redisHost := os.Getenv("REDISHOST")
	// redisPassword := os.Getenv("REDISPASSWORD")
	// redisPort := os.Getenv("REDISPORT")
	// port := os.Getenv("AUTHPORT")
	// dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))

	// p, err := utils.BrokerConnect(brokerURL)
	// if err != nil {
	// 	entities.MessageLogs.ErrorLog.Printf("Error connecting because of %s\n", err)
	// 	os.Exit(1)
	// }

	// c, err := utils.ConsumerConnect(brokerURL)
	// if err != nil {
	// 	entities.MessageLogs.ErrorLog.Printf("Error connecting because of %s\n", err)
	// 	os.Exit(1)
	// }

	// redis, err := repo.ConnectToRedis(ctx, redisHost, redisPassword, redisPort)
	// if err != nil {
	// 	entities.MessageLogs.ErrorLog.Printf("Error connecting redis because of %s\n", err)
	// 	os.Exit(1)
	// }

	// db, err := utils.DbConnect(dsn)
	// if err != nil {
	// 	entities.MessageLogs.ErrorLog.Printf("%s %s", entities.ErrorDBConnection.Error(), err)
	// 	os.Exit(1)
	// }

	// b.AuthPort = port
	// b.Cache = redis
	// b.KafkaProducer = p
	// b.KafkaConsumer = c
	// b.Broker = brokerURL
	// b.Topic = authTopic
	// b.Key = authKey
	// b.DB = db

	entities.MessageLogs.InfoLog.Printf("Connections done in %v\n", time.Since(startTime))

}
