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

func (b *Base) RabbitConsumer(wg *sync.WaitGroup) {
	defer wg.Done()

	ch, err := b.rabbitConn.Channel()
	if err != nil {
		os.Exit(1)
	}

	defer ch.Close()

	err = ch.Qos(1, 0, false)
	if err != nil {
		os.Exit(1)
	}

	_, err = ch.QueueDeclare(b.queueName, true, false, false, false, nil)
	if err != nil {
		os.Exit(1)
	}

	msgs, err := ch.Consume(b.queueName, "", false, false, false, false, nil)
	if err != nil {
		os.Exit(1)
	}

	wkWg := &sync.WaitGroup{}
	workers := 3

	for i := 0; i < workers; i++ {
		wkWg.Add(1)

		go func(workerId int) {
			defer wkWg.Done()

			for {
				select {
				case <-b.ctx.Done():
					log.Printf("Worker %d shutting down", workerId)
					return
				case msg, ok := <-msgs:
					if !ok {
						log.Printf("Worker %d channel closed", workerId)
						return
					}

					_ = msg
				}

			}

		}(i + 1)
	}

	<-b.ctx.Done()
	wkWg.Wait()
	log.Println("All consumer workers shut down")

}
