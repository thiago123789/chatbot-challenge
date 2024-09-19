package workflow

import (
	"code-challenge/pkg/openai"
	"context"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

// ChatBotQuestion is the input to the ChatBotWorkflow.
type ChatBotQuestion struct {
	User     string
	Question string
}

// ChatBotAnswer is the response from the ChatBotWorkflow.
type ChatBotAnswer struct {
	User   string
	Answer string
}

// ChatActivity is a Temporal activity that calls the OpenAI API to get an answer to a question.
func ChatActivity(ctx context.Context, question string) (*string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ChatActivity started.", "Question", question)

	ans, err := openai.GetAnswerFromGpt(question)

	if err != nil {
		logger.Error("Not able to retrieve answers from GPT.", "Error", err)
		return nil, err
	}

	return ans, nil
}

// ChatBotWorkflow is a Temporal workflow that orchestrates the ChatActivity to get an answer to a question.
func ChatBotWorkflow(ctx workflow.Context, input ChatBotQuestion) (*ChatBotAnswer, error) {
	// Define a retry policy for the workflow activities
	retryPolicy := &temporal.RetryPolicy{
		InitialInterval:        time.Second,       // Initial interval between retries
		BackoffCoefficient:     2.0,               // Exponential backoff coefficient
		MaximumInterval:        time.Second * 100, // Maximum interval between retries
		MaximumAttempts:        0,                 // Unlimited retry attempts
		NonRetryableErrorTypes: []string{},        // List of non-retryable error types
	}

	// Set activity options including timeouts and retry policy
	opts := workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,  // Timeout for each activity execution
		ScheduleToCloseTimeout: 180 * time.Second, // Total timeout for the activity
		RetryPolicy:            retryPolicy,       // Apply the defined retry policy
	}

	// Get a logger instance for the workflow context
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ChatBotWorkflow", "User", input.User, "Question", input.Question)

	// Apply the activity options to the workflow context
	ctx = workflow.WithActivityOptions(ctx, opts)

	var resultAnswer string
	// Execute the ChatActivity with the provided question and get the result
	err := workflow.ExecuteActivity(ctx, ChatActivity, input.Question).Get(ctx, &resultAnswer)

	if err != nil {
		// Log the error if the activity execution fails
		logger.Error("Activity failed.", "Error", err)
		return nil, err
	}

	// Log the successful completion of the workflow
	logger.Info("ChatBotWorkflow completed.", "User", input.User, "Answer", resultAnswer)

	// Create the workflow result with the user and the answer
	workflowResult := &ChatBotAnswer{
		User:   input.User,
		Answer: resultAnswer,
	}
	return workflowResult, nil
}
