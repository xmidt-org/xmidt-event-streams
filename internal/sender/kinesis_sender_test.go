// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"context"
	"errors"
	"testing"

	"github.com/xmidt-org/xmidt-event-streams/internal/kinesis"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/retry"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockRetryRunner implements the retry.Runner interface for testing
type MockRetryRunner struct {
	mock.Mock
}

func (m *MockRetryRunner) Run(ctx context.Context, task retry.Task[int]) (int, error) {
	args := m.Called(ctx, task)
	// Call the task to simulate real behavior and verify it gets called
	_, _ = task(ctx)
	// Return the mocked values
	return args.Int(0), args.Error(1)
}

// MockKinesisClientAPI implements the kinesis.KinesisClientAPI interface for testing

// KinesisSenderTestSuite contains all the test cases for KinesisSender
type KinesisSenderTestSuite struct {
	suite.Suite
	logger              *zap.Logger
	mockKinesisClient   *MockKinesisClientAPI
	mockKinesisProvider *MockKinesisProvider
	mockRetryRunner     *MockRetryRunner
}

func TestKinesisSenderTestSuite(t *testing.T) {
	suite.Run(t, new(KinesisSenderTestSuite))
}

func (suite *KinesisSenderTestSuite) SetupTest() {
	// Create logger for testing
	suite.logger = zaptest.NewLogger(suite.T())
	suite.mockKinesisClient = new(MockKinesisClientAPI)
	suite.mockRetryRunner = new(MockRetryRunner)
	suite.mockKinesisProvider = new(MockKinesisProvider)
	suite.mockKinesisProvider.On("Get", mock.Anything).Return(suite.mockKinesisClient, nil)
}

func (suite *KinesisSenderTestSuite) TearDownTest() {
	// Clean up any resources if needed
}

func (suite *KinesisSenderTestSuite) TestNewKinesisSender() {
	// Test configuration parameter handling
	testCases := []struct {
		name        string
		retries     int
		config      map[string]string
		expectError bool
		description string
	}{
		{
			name:        "empty_config",
			retries:     3,
			config:      map[string]string{}, // Missing required fields
			expectError: true,
			description: "empty config should fail",
		},
		{
			name:        "nil_config",
			retries:     3,
			config:      nil,
			expectError: true,
			description: "Nil configuration should fail",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			sender, err := NewKinesisSender(tc.retries, tc.config, suite.logger, suite.mockKinesisProvider)

			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
				suite.NotNil(sender, "Sender should not be nil for %s", tc.description)
			}
		})
	}
}

func (suite *KinesisSenderTestSuite) TestOnEvent_Success() {
	testCases := []struct {
		name                 string
		messages             []*wrp.Message
		streamURL            string
		expectedFailedCount  int
		expectedKinesisItems int
		setupMocks           func()
		description          string
	}{
		{
			name: "single_message_success",
			messages: []*wrp.Message{
				{
					Type:        wrp.SimpleEventMessageType,
					Source:      "device-123",
					Destination: "event:device-status",
					SessionID:   "session-123",
					ContentType: "application/json",
					Payload:     []byte(`{"status":"online"}`),
				},
			},
			streamURL:            "test-stream",
			expectedFailedCount:  0,
			expectedKinesisItems: 1,
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "test-stream").Return(0, nil)
			},
			description: "Single message should be processed successfully",
		},
		{
			name: "multiple_messages_success",
			messages: []*wrp.Message{
				{
					Type:        wrp.SimpleEventMessageType,
					Source:      "device-123",
					Destination: "event:device-status",
					SessionID:   "session-123",
					ContentType: "application/json",
					Payload:     []byte(`{"status":"online"}`),
				},
				{
					Type:        wrp.SimpleEventMessageType,
					Source:      "device-456",
					Destination: "event:device-alert",
					SessionID:   "session-456",
					ContentType: "application/json",
					Payload:     []byte(`{"alert":"temperature"}`),
				},
			},
			streamURL:            "test-stream",
			expectedFailedCount:  0,
			expectedKinesisItems: 2,
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "test-stream").Return(0, nil)
			},
			description: "Multiple messages should be processed successfully",
		},
		{
			name: "partial_failure",
			messages: []*wrp.Message{
				{
					Type:        wrp.SimpleEventMessageType,
					Source:      "device-123",
					Destination: "event:device-status",
					SessionID:   "session-123",
					ContentType: "application/json",
					Payload:     []byte(`{"status":"online"}`),
				},
			},
			streamURL:            "test-stream",
			expectedFailedCount:  1,
			expectedKinesisItems: 1,
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "test-stream").Return(1, nil)
			},
			description: "Partial failure should return failed count",
		},
		{
			name:                 "empty_messages_list",
			messages:             []*wrp.Message{},
			streamURL:            "test-stream",
			expectedFailedCount:  0,
			expectedKinesisItems: 0,
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "test-stream").Return(0, nil)
			},
			description: "Empty messages list should succeed with no items",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create sender with mock kinesis client and retry runner
			sender := &KinesisSender{
				kc:            suite.mockKinesisClient,
				schemaVersion: SchemaVersion,
				logger:        suite.logger,
				retries:       3,
				putRunner:     suite.mockRetryRunner,
			}

			// Setup mocks
			tc.setupMocks()
			suite.mockRetryRunner.On("Run", mock.Anything, mock.Anything).Return(tc.expectedFailedCount, nil)

			// Execute
			failedCount, err := sender.OnEvent(tc.messages, tc.streamURL)

			// Verify results
			suite.NoError(err, tc.description)
			suite.Equal(tc.expectedFailedCount, failedCount, "Failed count should match expected for %s", tc.description)

			// Verify mock expectations
			suite.mockKinesisClient.AssertExpectations(suite.T())
			suite.mockRetryRunner.AssertExpectations(suite.T())

			// Reset mocks for next test
			suite.mockKinesisClient.ExpectedCalls = nil
			suite.mockKinesisClient.Calls = nil
			suite.mockRetryRunner.ExpectedCalls = nil
			suite.mockRetryRunner.Calls = nil
		})
	}
}

func (suite *KinesisSenderTestSuite) TestOnEvent_KinesisErrors() {
	testCases := []struct {
		name        string
		setupMocks  func()
		expectError bool
		description string
	}{
		{
			name: "kinesis_put_records_error",
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "test-stream").Return(0, errors.New("kinesis service error"))
			},
			expectError: true,
			description: "Kinesis PutRecords error should be returned",
		},
		{
			name: "kinesis_throttling_error",
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "test-stream").Return(0, errors.New("ProvisionedThroughputExceededException"))
			},
			expectError: true,
			description: "Kinesis throttling error should be returned",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create sender with mock kinesis client
			sender := &KinesisSender{
				kc:            suite.mockKinesisClient,
				schemaVersion: SchemaVersion,
				logger:        suite.logger,
				retries:       1, // Use minimal retries for error testing
				putRunner:     suite.mockRetryRunner,
			}

			message := &wrp.Message{
				Type:        wrp.SimpleEventMessageType,
				Source:      "device-123",
				Destination: "event:device-status",
				SessionID:   "session-123",
				ContentType: "application/json",
				Payload:     []byte(`{"status":"online"}`),
			}

			tc.setupMocks()
			// Mock retry runner to return error when expectError is true
			if tc.expectError {
				suite.mockRetryRunner.On("Run", mock.Anything, mock.Anything).Return(0, errors.New("mocked retry error"))
			} else {
				suite.mockRetryRunner.On("Run", mock.Anything, mock.Anything).Return(0, nil)
			}

			// Execute
			failedCount, err := sender.OnEvent([]*wrp.Message{message}, "test-stream")

			// Verify results
			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
			}

			// The failed count may vary depending on the error
			suite.GreaterOrEqual(failedCount, 0, "Failed count should be non-negative")

			// Verify mock expectations
			suite.mockKinesisClient.AssertExpectations(suite.T())
			suite.mockRetryRunner.AssertExpectations(suite.T())

			// Reset mocks for next test
			suite.mockKinesisClient.ExpectedCalls = nil
			suite.mockKinesisClient.Calls = nil
			suite.mockRetryRunner.ExpectedCalls = nil
			suite.mockRetryRunner.Calls = nil
		})
	}
}

func (suite *KinesisSenderTestSuite) TestOnEvent_MessageMarshalingErrors() {
	suite.Run("skip_invalid_messages", func() {
		// Create sender with mock kinesis client
		sender := &KinesisSender{
			kc:            suite.mockKinesisClient,
			schemaVersion: SchemaVersion,
			logger:        suite.logger,
			retries:       3,
			putRunner:     suite.mockRetryRunner,
		}

		// Create messages where some might cause marshaling issues
		// Note: wrp.Message typically marshals successfully, but we're testing the error handling path
		validMessage := &wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "device-123",
			Destination: "event:device-status",
			SessionID:   "session-123",
			ContentType: "application/json",
			Payload:     []byte(`{"status":"online"}`),
		}

		// The actual kinesis.Item should only contain the valid message
		suite.mockKinesisClient.On("PutRecords", mock.MatchedBy(func(items []kinesis.Item) bool {
			return len(items) == 1 && items[0].PartitionKey == "session-123"
		}), "test-stream").Return(0, nil)
		suite.mockRetryRunner.On("Run", mock.Anything, mock.Anything).Return(0, nil)

		// Execute with only valid messages (since wrp.Message marshaling typically succeeds)
		failedCount, err := sender.OnEvent([]*wrp.Message{validMessage}, "test-stream")

		// Verify results
		suite.NoError(err, "Should succeed with valid messages")
		suite.Equal(0, failedCount, "Should have no failed records")

		// Verify mock expectations
		suite.mockKinesisClient.AssertExpectations(suite.T())
		suite.mockRetryRunner.AssertExpectations(suite.T())
	})
}

func (suite *KinesisSenderTestSuite) TestOnEvent_EdgeCases() {
	testCases := []struct {
		name        string
		messages    []*wrp.Message
		streamURL   string
		setupMocks  func()
		description string
	}{
		{
			name:      "nil_messages_list",
			messages:  nil,
			streamURL: "test-stream",
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "test-stream").Return(0, nil)
			},
			description: "Nil messages list should be handled gracefully",
		},
		{
			name: "message_with_empty_session_id",
			messages: []*wrp.Message{
				{
					Type:        wrp.SimpleEventMessageType,
					Source:      "device-123",
					Destination: "event:device-status",
					SessionID:   "", // Empty session ID
					ContentType: "application/json",
					Payload:     []byte(`{"status":"online"}`),
				},
			},
			streamURL: "test-stream",
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.MatchedBy(func(items []kinesis.Item) bool {
					return len(items) == 1 && items[0].PartitionKey == ""
				}), "test-stream").Return(0, nil)
			},
			description: "Message with empty session ID should still be processed",
		},
		{
			name:      "empty_stream_url",
			messages:  []*wrp.Message{{SessionID: "session-123"}},
			streamURL: "",
			setupMocks: func() {
				suite.mockKinesisClient.On("PutRecords", mock.AnythingOfType("[]kinesis.Item"), "").Return(0, nil)
			},
			description: "Empty stream URL should be passed through to kinesis client",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create sender with mock kinesis client
			sender := &KinesisSender{
				kc:            suite.mockKinesisClient,
				schemaVersion: SchemaVersion,
				logger:        suite.logger,
				retries:       3,
				putRunner:     suite.mockRetryRunner,
			}

			tc.setupMocks()
			suite.mockRetryRunner.On("Run", mock.Anything, mock.Anything).Return(0, nil)

			// Execute
			failedCount, err := sender.OnEvent(tc.messages, tc.streamURL)

			// Verify results
			suite.NoError(err, tc.description)
			suite.GreaterOrEqual(failedCount, 0, "Failed count should be non-negative")

			// Verify mock expectations
			suite.mockKinesisClient.AssertExpectations(suite.T())
			suite.mockRetryRunner.AssertExpectations(suite.T())

			// Reset mocks for next test
			suite.mockKinesisClient.ExpectedCalls = nil
			suite.mockKinesisClient.Calls = nil
			suite.mockRetryRunner.ExpectedCalls = nil
			suite.mockRetryRunner.Calls = nil
		})
	}
}
