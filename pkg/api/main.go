package main

import (
	codingchallenge "code-challenge/pkg/workflow"
	"context"
	"encoding/json"
	client2 "go.temporal.io/sdk/client"
	"log"
	"net/http"
	"os"
)

type ChatBotRequestInput struct {
	Question string `json:"question"`
	User     string `json:"user"`
}

// Handles the incoming request and executes the temporal workflow passing the question and user as input
func handler(w http.ResponseWriter, r *http.Request) {
	// Decode the incoming JSON request body into a ChatBotRequestInput struct
	var chatBotRequest ChatBotRequestInput
	_ = json.NewDecoder(r.Body).Decode(&chatBotRequest)

	// Initialize a new Temporal client with lazy loading
	client, err := client2.NewLazyClient(client2.Options{
		HostPort:  os.Getenv("TEMPORAL_HOST_PORT"),
		Namespace: os.Getenv("TEMPORAL_NAMESPACE"),
	})
	defer client.Close()

	// Check if there was an error initializing the Temporal client
	if err != nil {
		log.Fatal("Unable to initialize temporal client", err)
	}

	// Define workflow options including ID and TaskQueue
	wfOpts := client2.StartWorkflowOptions{
		ID:        "chat_bot_workflow_ID",
		TaskQueue: "chat_bot_workflow_task_queue",
	}

	// Execute the workflow with the provided question and user from the request
	wr, err := client.ExecuteWorkflow(context.Background(), wfOpts, codingchallenge.ChatBotWorkflow, codingchallenge.ChatBotQuestion{
		Question: chatBotRequest.Question,
		User:     chatBotRequest.User,
	})

	// Check if there was an error executing the workflow
	if err != nil {
		log.Fatal("Unable to execute workflow", err)
	}

	// Retrieve the result of the workflow execution
	var result *codingchallenge.ChatBotAnswer
	err = wr.Get(context.Background(), &result)

	// Check if there was an error getting the workflow result
	if err != nil {
		log.Fatalln("Unable to get workflow result", err)
	}

	// Set the response content type to application/json
	w.Header().Add("content-type", "application/json")

	// Encode the result into the response writer as JSON
	err = json.NewEncoder(w).Encode(result)

	// Check if there was an error encoding the response
	if err != nil {
		log.Fatalln("Error parting response to api")
		return
	}
}

// Starts the API server at port 3002
func main() {
	http.HandleFunc("/chat", handler)
	log.Println("Server started at http://localhost:3002")
	log.Fatal(http.ListenAndServe(":3002", nil))
}
