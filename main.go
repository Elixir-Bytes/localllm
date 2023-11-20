package main

import (
	"flag"
	"fmt"
)

const defaultHost = "http://localhost:11434"

var model = flag.String("model", "mistral", "model to use")
var prompt = flag.String("prompt", "", "prompt for model")

func main() {
	flag.Parse()
	request := request{
		Model:  *model,
		Prompt: *prompt,
	}

	makeRequest(request, fmt.Sprintf("%s/api/generate", defaultHost))
}
