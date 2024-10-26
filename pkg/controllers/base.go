package controllers

import (
	"os"
	"time"

	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Base struct {
	KafkaProducer *kafka.Producer
	KafkaConsumer *kafka.Consumer
}

func (b *Base) Init() {
	startTime := time.Now()

	p, err := utils.BrokerConnect("localhost:19092")
	if err != nil {
		utils.MessageLogs.ErrorLog.Printf("Error connecting because of %s\n", err)
		os.Exit(1)
	}

	b.KafkaProducer = p
	utils.MessageLogs.InfoLog.Printf("Producer connection done in %v\n", time.Since(startTime))

}
