package workflow

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"
	"testing"
)

func Test_ChatBotWorkflow_Success(t *testing.T) {
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	var returnValue *string
	value := "Paris"
	returnValue = &value

	env.OnActivity(ChatActivity, mock.Anything, "What is the capital of France?").Return(returnValue, nil)

	env.ExecuteWorkflow(ChatBotWorkflow, ChatBotQuestion{
		User:     "test_user",
		Question: "What is the capital of France?",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result ChatBotAnswer
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "test_user", result.User)
	assert.Equal(t, value, result.Answer)
}

func Test_ChatBotWorkflow_Activity_Failure(t *testing.T) {
	ts := testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	env.OnActivity(ChatActivity, mock.Anything, "What is the capital of France?").Return(nil, errors.New("API error"))

	env.ExecuteWorkflow(ChatBotWorkflow, ChatBotQuestion{
		User:     "test_user",
		Question: "What is the capital of France?",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.Error(t, env.GetWorkflowError())
}

func TestChatBotWorkflow_RetryPolicy(t *testing.T) {
	ts := testsuite.WorkflowTestSuite{}
	env := ts.NewTestWorkflowEnvironment()

	var returnValue *string
	value := "Paris"
	returnValue = &value

	env.OnActivity(ChatActivity, mock.Anything, "What is the capital of France?").Return(nil, errors.New("API error")).Times(3).Return(returnValue, nil)

	env.ExecuteWorkflow(ChatBotWorkflow, ChatBotQuestion{
		User:     "test_user",
		Question: "What is the capital of France?",
	})

	assert.True(t, env.IsWorkflowCompleted())
	assert.NoError(t, env.GetWorkflowError())

	var result ChatBotAnswer
	assert.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, "test_user", result.User)
	assert.Equal(t, value, result.Answer)
}
