package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

/*
## Parameters

model: (required) the model name
prompt: the prompt to generate a response for

Advanced parameters (optional):

format: the format to return a response in. Currently the only accepted value is json
options: additional model parameters listed in the documentation for the Modelfile such as temperature
system: system prompt to (overrides what is defined in the Modelfile)
template: the full prompt or prompt template (overrides what is defined in the Modelfile)
context: the context parameter returned from a previous request to /generate, this can be used to keep a short conversational memory
stream: if false the response will be returned as a single response object, rather than a stream of objects
raw: if true no formatting will be applied to the prompt and no context will be returned. You may choose to use the raw parameter if you are specifying a full templated prompt in your request to the API, and are managing history yourself.
*/

type request struct {
	ChatID string `json:"chat_id"`
	Model  string `json:"model"`
	Prompt string `json:"prompt"`

	Format   string  `json:"format,omitempty"`
	System   string  `json:"system,omitempty"`
	Template string  `json:"template,omitempty"`
	Context  []int64 `json:"context,omitempty"`
	Stream   bool    `json:"stream,omitempty"`
	Raw      bool    `json:"raw,omitempty"`
}

type response struct {
	ChatID          string  `json:"chat_id"`
	Model           string  `json:"model"`
	CreatedAt       string  `json:"created_at"`
	Response        string  `json:"response"`
	Done            bool    `json:"done"`
	Context         []int64 `json:"context"`
	TotalDuration   int64   `json:"total_duration"`
	LoadDuration    int64   `json:"load_duration"`
	PromptEvalCount int64   `json:"prompt_eval_count"`
	EvalCount       int64   `json:"eval_count"`
	EvalDuration    int64   `json:"eval_duration"`
}

func makeRequest(payload request, url string, responseChannel chan response) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(payload)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	handleBody(resp.Body, payload.ChatID, responseChannel)
}

// TODO: send errors over channel when there is an error
func handleBody(body io.ReadCloser, chatID string, responseChannel chan response) {
	scanner := bufio.NewScanner(body)
	var response response
	for scanner.Scan() {
		err := json.Unmarshal(scanner.Bytes(), &response)
		if err != nil {
			fmt.Fprintln(os.Stderr, "reading response:", err)
			continue
		}

		fmt.Printf("ChatID: %s => %v\n", chatID, response)
		response.ChatID = chatID
		responseChannel <- response
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}
