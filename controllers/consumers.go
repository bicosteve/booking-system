package controllers

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/pkg/utils"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func (b *Base) Consumer(wg *sync.WaitGroup, topic string) {
	defer wg.Done()

	consumer := b.KafkaConsumer

	err := consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		_resp := fmt.Sprintf("CONSUMER ERROR: problem %v:\n", err)
		utils.LogError(_resp, entities.ErrorLog)
		os.Exit(1)
	}

	// Context to handle signal interrupts
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle an Ctrl + C signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigchan
		cancel()
		os.Exit(1)
	}()

	for {
		select {
		case <-ctx.Done():
			consumer.Close()
			_msg := fmt.Sprintf("Detected termination signal %v: exiting\n", <-ctx.Done())
			utils.LogError(_msg, entities.ErrorLog)
			os.Exit(1)
		default:
			msg, err := consumer.ReadMessage(1000 * time.Millisecond)
			if err != nil {
				if err.(kafka.Error).IsTimeout() {
					continue
				}
				msg := fmt.Sprintf("Consumer error: %v %v:\n", err, msg)
				utils.LogError(msg, entities.ErrorLog)
				return

			}

			_imsg := fmt.Sprintf("Consumed from topic %s key=%-10s value = %s\n", *msg.TopicPartition.Topic, string(msg.Key), string(msg.Value))
			utils.LogInfo(_imsg, entities.InfoLog)
		}

	}

}

func (b *Base) RabbitMQConsumer(wg *sync.WaitGroup) {
	defer wg.Done()

	ch, err := b.rabbitConn.Channel()
	if err != nil {
		log.Fatal("Failed to open channel due to: " + err.Error())
		os.Exit(1)
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		b.queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive:false  multiple consumers can access this
		false, // no-wait : mq responds allowing error checking
		nil,
	)
	if err != nil {
		log.Fatal("Failed to declare queue due to: " + err.Error())
		os.Exit(1)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",    // consumer tag
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool)

	go func() {
		for data := range msgs {
			// 1. Insert into table

			// 2. Just print it in the meantime

			fmt.Println(data.Body)

			// 3. Acknowledge the message so that no data is lost
			data.Ack(false)

		}
	}()

	go func() {
		<-sigs
		utils.LogError("RABBITCONSUMER: Termination signal received. Exiting...", entities.ErrorLog)
		done <- true
	}()

	utils.LogInfo("RABBITCONSUMER: Listing to  `%s` queue", entities.InfoLog, b.queueName)

	// <-done

}
