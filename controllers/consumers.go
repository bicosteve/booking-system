package controllers

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func (b *Base) Consumer(wg *sync.WaitGroup, topic string) {
	defer wg.Done()

	consumer := b.KafkaConsumer

	err := consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		entities.MessageLogs.InfoLog.Printf("CONSUMER ERROR: problem %v:\n", err)
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
			entities.MessageLogs.InfoLog.Printf("Detected termination signal %v: exiting\n", <-ctx.Done())
			os.Exit(1)
		default:
			msg, err := consumer.ReadMessage(1000 * time.Millisecond)
			if err != nil {
				if err.(kafka.Error).IsTimeout() {
					// utils.MessageLogs.InfoLog.Println("No new messages ... ")
					continue
				}
				entities.MessageLogs.ErrorLog.Printf("Consumer error: %v %v:\n", err, msg)
				return

			}

			entities.MessageLogs.InfoLog.Printf("Consumed from topic %s key=%-10s value = %s\n", *msg.TopicPartition.Topic, string(msg.Key), string(msg.Value))
		}

	}

}
