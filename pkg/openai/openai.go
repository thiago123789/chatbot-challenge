package openai

import (
	"context"
	"fmt"
	openai2 "github.com/sashabaranov/go-openai"
	"os"
)

// GetAnswerFromGpt calls ChatGPT api using GPT-4 to get an advanced answer to the user's question
func GetAnswerFromGpt(question string) (*string, error) {
	// Create a new client instance using the provided API key
	client := openai2.NewClient(os.Getenv("OPENAI_API_KEY"))

	// Make a request to the OpenAI API to create a chat completion
	// The request includes the model to use and the user's question
	resp, err := client.CreateChatCompletion(context.Background(), openai2.ChatCompletionRequest{
		Model: openai2.GPT4o20240513,
		Messages: []openai2.ChatCompletionMessage{{
			Role:    openai2.ChatMessageRoleUser,
			Content: question,
		}},
	})

	// Check if there was an error during the API call
	if err != nil {
		// Print the error to the console and return nil along with the error
		fmt.Println("Error: ", err)
		return nil, err
	}

	// Return the content of the first choice in the response message
	return &resp.Choices[0].Message.Content, nil
}
