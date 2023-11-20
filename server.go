package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rabbitMQEnvName   = "RABBITMQ_ADDR"
	consumerQueueName = "llm_jobs"
	resultsQueueName  = "llm_responses"
)

/*
Sample:
```json

	{
	  "chat_id": "100",
	  "model": "mistral",
	  "prompt": "what is your name?",
	}

```
*/
type Job struct {
	ChatID         string  `json:"chat_id"`
	Model          string  `json:"model"`
	Prompt         string  `json:"prompt"`
	Context        []int64 `json:"context"`
	PromptTemplate string  `json:"prompt_template"`
	SystemPrompt   string  `json:"system_prompt"`
}

type MessageServer struct {
	JobsConsumer     chan Job
	ResponseProducer chan response
	ollamaHost       string
	amqpChannel      *amqp.Channel
}

func NewMessageServer(ollamaHost string) *MessageServer {
	return &MessageServer{
		ollamaHost:       ollamaHost,
		JobsConsumer:     make(chan Job),
		ResponseProducer: make(chan response),
	}
}

func (m *MessageServer) Listen() error {
	fmt.Printf("MessageServer is listening...\n")
	rabbitMQAddress := os.Getenv(rabbitMQEnvName)
	conn, err := amqp.Dial(rabbitMQAddress)

	if err != nil {
		return err
	}

	channel, err := conn.Channel()

	if err != nil {
		return err
	}

	m.amqpChannel = channel

	queue, err := channel.QueueDeclare(
		consumerQueueName, // name
		false,             // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)

	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare(
		resultsQueueName, // name
		false,            // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)

	if err != nil {
		return err
	}

	msgs, err := channel.Consume(
		queue.Name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)

	if err != nil {
		return err
	}

	go func() {
		defer conn.Close()
		defer channel.Close()

		var job Job
		var err error

		for msg := range msgs {
			err = json.Unmarshal(msg.Body, &job)

			if err != nil {
				log.Printf("Error with job JSON: %s => %v\n", err, msg.Body)
				continue
			}

			m.JobsConsumer <- job
		}
	}()

	go func() {
		for response := range m.ResponseProducer {
			m.sendEvent(response)
		}
	}()

	return nil
}

func (messageServer *MessageServer) sendEvent(response response) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	jsonBytes, err := json.Marshal(response)

	if err != nil {
		fmt.Printf("Failed to serialze response '%s': %#v\n", err, response)
		return
	}

	err = messageServer.amqpChannel.PublishWithContext(ctx,
		"",               // exchange
		resultsQueueName, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonBytes,
		})

	if err != nil {
		fmt.Printf("Failed to publish a message from response %#v\n", response)
		return
	}
}

func (server *MessageServer) handleJob(job Job) {
	request := request{
		ChatID: job.ChatID,
		Model:  job.Model,
		Prompt: job.Prompt,
	}

	url := fmt.Sprintf("%s/api/generate", server.ollamaHost)

	makeRequest(request, url, server.ResponseProducer)
}

func (server *MessageServer) StartWorker() {
	go func() {
		for receivedJob := range server.JobsConsumer {
			var wg sync.WaitGroup
			go func(job Job) {
				wg.Add(1)
				server.handleJob(job)
				wg.Done()
			}(receivedJob)
			wg.Wait()
		}
	}()
}
