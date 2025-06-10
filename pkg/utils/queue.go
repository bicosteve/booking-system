package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bicosteve/booking-system/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/streadway/amqp"
)

func ProducerConnect(brokerString string) (*kafka.Producer, error) {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		LogError("PRODUCER: Received termination signal. Exiting", entities.ErrorLog)
		os.Exit(1)

	}()

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokerString,
		"acks":              "all",
	})

	if err != nil {
		LogError("PRODUCER: Could not connect to broker becasue: "+err.Error(), entities.ErrorLog)
		return nil, err
	}

	LogInfo("PRODUCER: connected successfully", entities.InfoLog)

	return p, nil
}

func ConsumerConnect(broker string) (*kafka.Consumer, error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		LogError("CONSUMER: Received termination signal. Exiting", entities.ErrorLog)
		os.Exit(1)

	}()

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "kafka-go-getting-started",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		LogError("CONSUMER: Could not connect due to "+err.Error(), entities.ErrorLog)
		return nil, err
	}

	LogInfo("CONSUMER: connected successfully", entities.InfoLog)

	return c, nil
}

func QPublishMessage(broker, topic, key string, data any) error {
	wg := &sync.WaitGroup{}
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"acks":              "all",
	})

	if err != nil {
		LogError(err.Error(), entities.ErrorLog)
		return errors.New(err.Error())
	}

	defer p.Flush(15 * 100)
	defer p.Close()

	wg.Add(1)
	go func(w *sync.WaitGroup) {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					LogError("message cannot be delivered because of "+ev.TopicPartition.String(), entities.ErrorLog)
				} else {
					_msg := fmt.Sprintf("Produced events to topic %s key = %-10s value = %s\n", *ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))

					LogInfo(_msg, entities.InfoLog)

				}
			}
		}
		w.Done()
	}(wg)

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: -1},
		Key:            []byte(key),
		Value:          []byte(string(dataBytes)),
	}, nil)

	return nil
}

// RabbitMQ Section

var RabbitMQClient *entities.RabbitMQ

func NewRabbitMQConnection(qURI string) (*amqp.Connection, error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		LogError("RABBITMQ: Termination signal. Exiting...", entities.ErrorLog)
		os.Exit(1)

	}()

	// 1. Connect to rabbitmq
	conn, err := amqp.Dial(qURI)
	if err != nil {
		LogError("RABBITMQ: Failed to connect due to: %s", entities.ErrorLog, err)
		log.Fatalf("RABBITMQ: Failed to connect due to: %s", err)
		return nil, err
	}

	// 2. Open a rabbitmq channel
	ch, err := conn.Channel()
	if err != nil {
		LogError("RABBITMQ: Failed to open a channel : %s", entities.ErrorLog, err)
		log.Fatalf("RABBITMQ: Failed to open a channel : %s", err)
		return nil, err
	}

	// 3. Store the connection & channel
	RabbitMQClient = &entities.RabbitMQ{
		Connection: conn,
		Channel:    ch,
	}

	LogInfo("RABBITMQ: Connected successfully", entities.InfoLog)

	return RabbitMQClient.Connection, nil
}

// Send messages to RabbitMQ
func PublishToMQ(queue string, data any) error {

	var rabbitMQ entities.RabbitMQ

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		LogError("RABBITMQ PUBLISHER: Termination signal. Exiting...", entities.ErrorLog)
		os.Exit(1)

	}()

	// 1. Declare a que to ensure it exists
	q, err := rabbitMQ.Channel.QueueDeclare(
		queue, // queue name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return err
	}

	// 2. convert body to json format
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 3. Publish the message to queue
	err = rabbitMQ.Channel.Publish(
		"",     // Exchange
		q.Name, // routing key, here queue name
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		},
	)

	if err != nil {
		return err
	}

	LogInfo("RABBITMQ: Message sent queue %s", entities.InfoLog, body)

	return nil
}
