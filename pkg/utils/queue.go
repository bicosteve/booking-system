package utils

import (
	"encoding/json"
	"errors"
	"fmt"
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

func ConnecRabbitMQBroker(qURI string) (*amqp.Connection, error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		LogError("RABBIT BROKER: Termination signal. Exiting...", entities.ErrorLog)
		os.Exit(1)

	}()

	conn, err := amqp.Dial(qURI)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func PublishToMQ(queue, ContentType string, data any, conn *amqp.Connection, channel *amqp.Channel) error {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		LogError("MQ PUBLISHER: Termination signal. Exiting...", entities.ErrorLog)
		os.Exit(1)

	}()

	// 1. Create a channel
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	defer ch.Close()

	// 2. Declare a queue
	_, err = ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	_data, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// 3. Publish the message
	err = ch.Publish(
		"",
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType: ContentType,
			Body:        []byte(_data),
		},
	)

	if err != nil {
		return err
	}

	return nil
}
