package main

import (
	codingchallenge "code-challenge/pkg/workflow"
	client2 "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"log"
	"os"
)

// Starts the worker that listens to the task queue "chat_bot_workflow_task_queue"
func main() {
	// Dial creates a new Temporal client with the provided options
	client, err := client2.Dial(client2.Options{
		HostPort:  os.Getenv("TEMPORAL_HOST_PORT"),
		Namespace: os.Getenv("TEMPORAL_NAMESPACE"),
	})
	defer client.Close() // Ensure the client is closed when the function exits

	// Check if there was an error initializing the Temporal client
	if err != nil {
		log.Fatalln("Unable to initialize client", err)
	}

	// Create a new worker that listens to the specified task queue
	w := worker.New(client, "chat_bot_workflow_task_queue", worker.Options{})

	// Register the ChatBotWorkflow with the worker
	w.RegisterWorkflow(codingchallenge.ChatBotWorkflow)

	// Register the ChatActivity with the worker
	w.RegisterActivity(codingchallenge.ChatActivity)

	// Run the worker and listen for interrupt signals
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("Unable to start worker", err)
	}
}
