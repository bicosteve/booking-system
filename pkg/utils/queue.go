package utils

import (
	"encoding/json"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bicosteve/booking-system/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func ProducerConnect(brokerString string) (*kafka.Producer, error) {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		entities.MessageLogs.InfoLog.Printf("PRODUCER: Received termination signal. Exiting")
		os.Exit(1)

	}()

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokerString,
		"acks":              "all",
	})

	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("PRODUCER: Could not connect to broker becasue:  %v\n", err)
		return nil, err
	}

	entities.MessageLogs.InfoLog.Println("PRODUCER: connected successfully")

	return p, nil
}

func ConsumerConnect(broker string) (*kafka.Consumer, error) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		entities.MessageLogs.InfoLog.Printf("CONSUMER: Received termination signal. Exiting")
		os.Exit(1)

	}()

	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "kafka-go-getting-started",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("CONSUMER: Could not connect to consumer because: %v\n", err)
		return nil, err
	}

	entities.MessageLogs.InfoLog.Println("CONSUMER: connected successfully")

	return c, nil
}

func QPublishMessage(broker, topic, key string, data any) error {
	wg := &sync.WaitGroup{}
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"acks":              "all",
	})

	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
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
					entities.MessageLogs.ErrorLog.Printf("Message not delivered because of %v\n ", ev.TopicPartition)
				} else {
					entities.MessageLogs.InfoLog.Printf("Produced events to topic %s key = %-10s value = %s\n", *ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))

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
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          []byte(string(dataBytes)),
	}, nil)

	return nil
}
