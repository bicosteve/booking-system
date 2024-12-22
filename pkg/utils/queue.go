package utils

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/bicosteve/booking-system/pkg/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func SendMessageToKafka(broker, topic, key string, data any) error {
	wg := &sync.WaitGroup{}
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"acks":              "all",
	})

	if err != nil {
		entities.MessageLogs.ErrorLog.Println(err)
		return errors.New(err.Error())
	}

	defer p.Close()
	defer p.Flush(15 * 1000)

	wg.Add(1)
	go func(w *sync.WaitGroup) {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					entities.MessageLogs.ErrorLog.Printf("Message deliver because of %v\n ", ev.TopicPartition)
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

func BrokerConnect(brokerString string) (*kafka.Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokerString,
		"acks":              "all",
	})

	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("BROKER ERROR: Could not connect broker becasue:  %v\n", err)
		return nil, err
	}

	entities.MessageLogs.InfoLog.Println("Broker connected successfully")

	return p, nil
}

func ConsumerConnect(broker string) (*kafka.Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"group.id":          "kafka-go-getting-started",
		"auto.offset.reset": "earliest",
	})

	if err != nil {
		entities.MessageLogs.ErrorLog.Printf("CONSUMER ERROR: Could not create consumer because: %v\n", err)
		return nil, err
	}

	entities.MessageLogs.InfoLog.Println("CONSUMER INFO: connected to broker")

	return c, nil
}
