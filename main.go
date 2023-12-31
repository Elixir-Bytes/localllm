package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var (
	ollamaHost  = flag.String("ollama-host", "http://localhost:11434", "Address of ollama server")
	workerCount = flag.Int("worker-count", 1, "How many processes should run in parallel")
)

func main() {
	flag.Parse()

	messageServer := NewMessageServer(*ollamaHost)

	fmt.Println("Starting message server...")
	err := messageServer.Listen()

	if err != nil {
		log.Fatalf("Error starting message server: %s\n", err)
	}

	for i := 0; i < *workerCount; i++ {
		fmt.Printf("Starting worker %d\n", i+1)
		messageServer.StartWorker()
	}

	waitForSignal()
}

func waitForSignal() {
	signal_chan := make(chan os.Signal, 1)

	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exit_chan := make(chan int)
	go func() {
		for {
			s := <-signal_chan
			switch s {
			// kill -SIGHUP XXXX
			case syscall.SIGHUP:
				fmt.Println("hungup")

			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				fmt.Println("\nExiting...")
				exit_chan <- 0

			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				fmt.Println("force stop")
				exit_chan <- 0

			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				fmt.Println("stop and core dump")
				exit_chan <- 0

			default:
				fmt.Println("Unknown signal.")
				exit_chan <- 1
			}
		}
	}()

	code := <-exit_chan
	os.Exit(code)
}
