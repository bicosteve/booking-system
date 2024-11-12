package controllers

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bicosteve/booking-system/pkg/utils"
)

func (b *Base) AuthConsumer(wg *sync.WaitGroup) {
	defer wg.Done()

	consumer := b.KafkaConsumer
	topic := b.Topic

	err := consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		utils.MessageLogs.InfoLog.Printf("CONSUMER ERROR: problem %v:\n", err)
		os.Exit(1)
	}

	// Handle an Ctrl + C signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	run := true
	for run {
		select {
		case sig := <-sigchan:
			utils.MessageLogs.InfoLog.Printf("Detected signal %v: terminating\n", sig)
			run = false
			os.Exit(1)

		default:
			msg, err := consumer.ReadMessage(1000 * time.Millisecond)
			if err != nil {
				// Just for information since consumer handles the errors
				utils.MessageLogs.ErrorLog.Printf("Consumer error: %v %v:\n", err, msg)
				continue
			}

			/*
				fmt.Printf("Consumed from topic %s key=%-10s value = %s\n", *ev.TopicPartition.Topic, string(ev.Key), string(ev.Value))
			*/

			utils.MessageLogs.InfoLog.Printf("Consumed from topic %s key=%-10s value = %s\n", *msg.TopicPartition.Topic, string(msg.Key), string(msg.Value))
		}

	}

	consumer.Close()

}
