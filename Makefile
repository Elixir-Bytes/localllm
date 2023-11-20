bin/localllm: clean
	go build -o bin/localllm ./...

clean:
	rm -rf bin
