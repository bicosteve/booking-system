package utils

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func SendMessageToKafka(broker, topic, key string, data any) error {
	wg := &sync.WaitGroup{}
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"acks":              "all",
	})

	if err != nil {
		MessageLogs.ErrorLog.Println(err)
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
					MessageLogs.ErrorLog.Printf("Message deliver because of %v\n ", ev.TopicPartition)
				} else {
					MessageLogs.InfoLog.Printf("Produced events to topic %s key = %-10s value = %s\n", *ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))

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
		return nil, err
	}

	return p, nil
}
