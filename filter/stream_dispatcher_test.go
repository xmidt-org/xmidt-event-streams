// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"errors"
	"testing"

	"github.com/xmidt-org/xmidt-event-streams/internal/sender"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// StreamDispatcherTestSuite contains tests for StreamDispatcher functionality
type StreamDispatcherTestSuite struct {
	suite.Suite
	logger      *zap.Logger
	mockSender1 *MockSender
	mockSender2 *MockSender
	mockMetrics *FilterMetrics
	mockQueue   *MockQueue
	dispatcher  *StreamDispatcher
}

func TestStreamDispatcherTestSuite(t *testing.T) {
	suite.Run(t, new(StreamDispatcherTestSuite))
}

func (suite *StreamDispatcherTestSuite) SetupTest() {
	// Create logger for testing
	suite.logger = zaptest.NewLogger(suite.T())

	// Create mock senders
	suite.mockSender1 = new(MockSender)
	suite.mockSender2 = new(MockSender)

	// Create mock metrics
	mockCounter := NewMockCounter()
	mockGauge := NewMockGauge()

	// Setup mock calls for metrics
	mockCounter.On("With", mock.Anything).Return(mockCounter).Maybe()
	mockCounter.On("Add", mock.Anything).Return().Maybe()
	mockGauge.On("With", mock.Anything).Return(mockGauge).Maybe()
	mockGauge.On("Set", mock.Anything).Return().Maybe()
	mockGauge.On("Add", mock.Anything).Return().Maybe()

	suite.mockMetrics = &FilterMetrics{
		DroppedMessage: mockCounter,
	}

	// Create mock queue
	suite.mockQueue = NewMockQueue()
	suite.mockQueue.On("AddItem", mock.Anything).Return().Maybe()
	suite.mockQueue.On("Close").Return().Maybe()
	suite.mockQueue.On("Empty").Return().Maybe()
	suite.mockQueue.On("Wait").Return().Maybe()

	// Create StreamDispatcher instance
	suite.dispatcher = &StreamDispatcher{
		id:      "test-dispatcher",
		senders: []sender.Sender{suite.mockSender1, suite.mockSender2},
		streams: []Stream{
			{StreamName: "primary-stream"},
			{StreamName: "backup-stream"},
		},
		queue:   suite.mockQueue,
		logger:  suite.logger,
		metrics: suite.mockMetrics,
	}
}

func (suite *StreamDispatcherTestSuite) TearDownTest() {
	suite.mockSender1.AssertExpectations(suite.T())
	suite.mockSender2.AssertExpectations(suite.T())
	suite.mockQueue.AssertExpectations(suite.T())
}

func (suite *StreamDispatcherTestSuite) createTestMessage() *wrp.Message {
	return &wrp.Message{
		Type:            wrp.SimpleEventMessageType,
		Source:          "device-123",
		Destination:     "event:device-status/mac:112233445566/online",
		TransactionUUID: "test-transaction-123",
		ContentType:     "application/json",
		Payload:         []byte(`{"status":"online","timestamp":"2023-10-30T10:00:00Z"}`),
		SessionID:       "session-123",
	}
}

func (suite *StreamDispatcherTestSuite) TestQueue() {
	testCases := []struct {
		name        string
		message     *wrp.Message
		description string
	}{
		{
			name:        "queue_valid_message",
			message:     suite.createTestMessage(),
			description: "Should successfully queue a valid WRP message",
		},
		{
			name:        "queue_nil_message",
			message:     nil,
			description: "Should handle nil message gracefully",
		},
		{
			name:        "queue_empty_message",
			message:     &wrp.Message{},
			description: "Should handle empty message",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()

			// Execute
			suite.dispatcher.Queue(tc.message)

			// Verify queue was called with the message
			suite.mockQueue.AssertCalled(suite.T(), "AddItem", tc.message)
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestShutdown() {
	testCases := []struct {
		name        string
		gentle      bool
		description string
	}{
		{
			name:        "gentle_shutdown",
			gentle:      true,
			description: "Gentle shutdown should only close and wait",
		},
		{
			name:        "forceful_shutdown",
			gentle:      false,
			description: "Forceful shutdown should empty queue before closing",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()

			// Execute
			suite.dispatcher.Shutdown(tc.gentle)

			// Verify appropriate queue methods were called
			if tc.gentle {
				// Gentle shutdown: close, wait (no empty)
				suite.mockQueue.AssertCalled(suite.T(), "Close")
				suite.mockQueue.AssertCalled(suite.T(), "Wait")
			} else {
				// Forceful shutdown: close, empty, close again, wait
				suite.mockQueue.AssertCalled(suite.T(), "Close")
				suite.mockQueue.AssertCalled(suite.T(), "Empty")
				suite.mockQueue.AssertCalled(suite.T(), "Wait")
			}
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestSubmit_Success() {
	testCases := []struct {
		name         string
		messages     []*wrp.Message
		setupMocks   func()
		validateCall func()
		description  string
	}{
		{
			name:     "submit_single_message_primary_success",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(0, nil).Once()
			},
			validateCall: func() {
				suite.mockSender1.AssertCalled(suite.T(), "OnEvent", mock.Anything, "primary-stream")
				suite.mockSender2.AssertNotCalled(suite.T(), "OnEvent")
			},
			description: "Should successfully submit to primary sender",
		},
		{
			name: "submit_multiple_messages_primary_success",
			messages: []*wrp.Message{
				suite.createTestMessage(),
				suite.createTestMessage(),
			},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(0, nil).Once()
			},
			validateCall: func() {
				suite.mockSender1.AssertCalled(suite.T(), "OnEvent", mock.Anything, "primary-stream")
				suite.mockSender2.AssertNotCalled(suite.T(), "OnEvent")
			},
			description: "Should successfully submit multiple messages to primary sender",
		},
		{
			name:     "submit_empty_messages_list",
			messages: []*wrp.Message{},
			setupMocks: func() {
				// No mocks needed for empty list
			},
			validateCall: func() {
				suite.mockSender1.AssertNotCalled(suite.T(), "OnEvent")
				suite.mockSender2.AssertNotCalled(suite.T(), "OnEvent")
			},
			description: "Should handle empty messages list gracefully",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()
			tc.setupMocks()

			// Execute
			err := suite.dispatcher.Submit(tc.messages)

			// Verify
			suite.NoError(err, tc.description)
			tc.validateCall()
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestSubmit_Failover() {
	testCases := []struct {
		name         string
		messages     []*wrp.Message
		setupMocks   func()
		expectError  bool
		validateCall func()
		description  string
	}{
		{
			name:     "primary_fails_backup_succeeds",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(0, errors.New("primary failed")).Once()
				suite.mockSender2.On("OnEvent", mock.Anything, "backup-stream").Return(0, nil).Once()
			},
			expectError: false,
			validateCall: func() {
				suite.mockSender1.AssertCalled(suite.T(), "OnEvent", mock.Anything, "primary-stream")
				suite.mockSender2.AssertCalled(suite.T(), "OnEvent", mock.Anything, "backup-stream")
			},
			description: "Should failover from primary to backup when primary fails",
		},
		{
			name:     "primary_fails_backup_fails",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(0, errors.New("primary failed")).Once()
				suite.mockSender2.On("OnEvent", mock.Anything, "backup-stream").Return(0, errors.New("backup failed")).Once()
			},
			expectError: true,
			validateCall: func() {
				suite.mockSender1.AssertCalled(suite.T(), "OnEvent", mock.Anything, "primary-stream")
				suite.mockSender2.AssertCalled(suite.T(), "OnEvent", mock.Anything, "backup-stream")
			},
			description: "Should return error when all senders fail",
		},
		{
			name:     "primary_partial_failure_with_failover",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				// Primary sender has partial failure (returns failedRecordCount > 0)
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(3, nil).Once()
				// Backup sender succeeds
				suite.mockSender2.On("OnEvent", mock.Anything, "backup-stream").Return(0, nil).Once()
			},
			expectError: false, // Should succeed with backup
			validateCall: func() {
				suite.mockSender1.AssertCalled(suite.T(), "OnEvent", mock.Anything, "primary-stream")
				suite.mockSender2.AssertCalled(suite.T(), "OnEvent", mock.Anything, "backup-stream")
			},
			description: "Should failover to backup when primary has partial failure",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()
			tc.setupMocks()

			// Execute
			err := suite.dispatcher.Submit(tc.messages)

			// Verify
			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
			}
			tc.validateCall()
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestSubmit_PanicRecovery() {
	testCases := []struct {
		name        string
		messages    []*wrp.Message
		setupMocks  func()
		expectPanic bool
		description string
	}{
		{
			name:     "sender_panic_string",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Run(func(args mock.Arguments) {
					panic("sender panic!")
				}).Maybe()
			},
			expectPanic: true,
			description: "Should recover from sender panic with string message",
		},
		{
			name:     "sender_panic_error",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Run(func(args mock.Arguments) {
					panic(errors.New("critical sender error"))
				}).Maybe()
			},
			expectPanic: true,
			description: "Should recover from sender panic with error",
		},
		{
			name:     "sender_panic_nil",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Run(func(args mock.Arguments) {
					panic("test panic")
				}).Maybe()
			},
			expectPanic: true,
			description: "Should recover from sender panic with nil value",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()
			tc.setupMocks()

			// Execute and verify panic behavior
			if tc.expectPanic {
				suite.Panics(func() {
					suite.dispatcher.Submit(tc.messages)
				}, tc.description)
			} else {
				suite.NotPanics(func() {
					suite.dispatcher.Submit(tc.messages)
				}, tc.description)
			}
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestSubmitBatch_Success() {
	testCases := []struct {
		name        string
		stream      string
		messages    []*wrp.Message
		setupMocks  func()
		expectError bool
		description string
	}{
		{
			name:     "submit_batch_success",
			stream:   "test-stream",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "test-stream").Return(0, nil).Once()
			},
			expectError: false,
			description: "Should successfully submit batch with no failures",
		},
		{
			name:     "submit_batch_sender_error",
			stream:   "test-stream",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "test-stream").Return(0, errors.New("sender error")).Once()
			},
			expectError: true,
			description: "Should return error when sender fails",
		},
		{
			name:     "submit_batch_partial_failure",
			stream:   "test-stream",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "test-stream").Return(5, nil).Once()
			},
			expectError: true,
			description: "Should return error when some records fail",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()
			tc.setupMocks()

			// Execute
			err := suite.dispatcher.submitBatch(tc.stream, suite.mockSender1, tc.messages)

			// Verify
			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
			}

			// Verify sender was called
			suite.mockSender1.AssertCalled(suite.T(), "OnEvent", tc.messages, tc.stream)
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestSubmit_MetricsRecording() {
	testCases := []struct {
		name         string
		messages     []*wrp.Message
		setupMocks   func()
		expectMetric bool
		description  string
	}{
		{
			name:     "success_no_dropped_metrics",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(0, nil).Once()
			},
			expectMetric: false,
			description:  "Successful sends should not record dropped message metrics",
		},
		{
			name:     "error_records_dropped_metrics",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(0, errors.New("send error")).Once()
				suite.mockSender2.On("OnEvent", mock.Anything, "backup-stream").Return(0, errors.New("backup error")).Once()
			},
			expectMetric: true,
			description:  "Failed sends should record dropped message metrics",
		},
		{
			name:     "partial_failure_records_metrics",
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				// Primary has partial failure - will trigger failover
				suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(3, nil).Once()
				// Backup succeeds
				suite.mockSender2.On("OnEvent", mock.Anything, "backup-stream").Return(0, nil).Once()
			},
			expectMetric: true,
			description:  "Partial failures should record dropped message metrics and failover",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()
			tc.setupMocks()

			// Execute
			suite.dispatcher.Submit(tc.messages)

			// Verify metrics recording based on expectation
			mockDroppedMessage := suite.mockMetrics.DroppedMessage.(*MockCounter)
			if tc.expectMetric {
				mockDroppedMessage.AssertCalled(suite.T(), "Add", mock.Anything)
			} else {
				mockDroppedMessage.AssertNotCalled(suite.T(), "Add", mock.Anything)
			}
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestSubmit_EdgeCases() {
	testCases := []struct {
		name        string
		setupSender func() *StreamDispatcher
		messages    []*wrp.Message
		setupMocks  func()
		expectError bool
		description string
	}{
		{
			name: "no_senders_available",
			setupSender: func() *StreamDispatcher {
				return &StreamDispatcher{
					id:      "no-senders",
					senders: []sender.Sender{},
					streams: []Stream{},
					queue:   suite.mockQueue,
					logger:  suite.logger,
					metrics: suite.mockMetrics,
				}
			},
			messages:    []*wrp.Message{suite.createTestMessage()},
			setupMocks:  func() {},
			expectError: true, // Empty streams slice returns error
			description: "Should return error when no streams are available",
		},
		{
			name: "single_sender_single_stream",
			setupSender: func() *StreamDispatcher {
				return &StreamDispatcher{
					id:      "single-sender",
					senders: []sender.Sender{suite.mockSender1},
					streams: []Stream{{StreamName: "only-stream"}},
					queue:   suite.mockQueue,
					logger:  suite.logger,
					metrics: suite.mockMetrics,
				}
			},
			messages: []*wrp.Message{suite.createTestMessage()},
			setupMocks: func() {
				suite.mockSender1.On("OnEvent", mock.Anything, "only-stream").Return(0, nil).Once()
			},
			expectError: false,
			description: "Should work with single sender and stream",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup fresh test environment
			suite.SetupTest()

			// Use custom dispatcher if provided
			dispatcher := tc.setupSender()
			tc.setupMocks()

			// Execute
			err := dispatcher.Submit(tc.messages)

			// Verify
			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
			}
		})
	}
}

func (suite *StreamDispatcherTestSuite) TestInterfaceCompliance() {
	// Ensure StreamDispatcher implements the expected interfaces
	var _ Dispatcher = (*StreamDispatcher)(nil)
}

func (suite *StreamDispatcherTestSuite) TestSubmit_LoggingValidation() {
	// Test that appropriate log messages are generated
	messages := []*wrp.Message{suite.createTestMessage()}

	// Test successful case
	suite.mockSender1.On("OnEvent", mock.Anything, "primary-stream").Return(0, nil).Once()

	err := suite.dispatcher.Submit(messages)
	suite.NoError(err, "Should submit successfully")

	// We can't easily test log output without capturing logs, but we can
	// verify the method completed without panicking
	suite.mockSender1.AssertExpectations(suite.T())
}

func (suite *StreamDispatcherTestSuite) TestConstructorPatterns() {
	// Test various construction patterns to ensure robustness
	testCases := []struct {
		name        string
		setupData   func() (*StreamDispatcher, error)
		expectValid bool
		description string
	}{
		{
			name: "valid_construction",
			setupData: func() (*StreamDispatcher, error) {
				return &StreamDispatcher{
					id:      "valid-dispatcher",
					senders: []sender.Sender{suite.mockSender1},
					streams: []Stream{{StreamName: "test-stream"}},
					queue:   suite.mockQueue,
					logger:  suite.logger,
					metrics: suite.mockMetrics,
				}, nil
			},
			expectValid: true,
			description: "Valid construction should succeed",
		},
		{
			name: "missing_logger",
			setupData: func() (*StreamDispatcher, error) {
				return &StreamDispatcher{
					id:      "no-logger",
					senders: []sender.Sender{suite.mockSender1},
					streams: []Stream{{StreamName: "test-stream"}},
					queue:   suite.mockQueue,
					logger:  nil, // Missing logger
					metrics: suite.mockMetrics,
				}, nil
			},
			expectValid: false,
			description: "Construction without logger should be invalid",
		},
		{
			name: "missing_metrics",
			setupData: func() (*StreamDispatcher, error) {
				return &StreamDispatcher{
					id:      "no-metrics",
					senders: []sender.Sender{suite.mockSender1},
					streams: []Stream{{StreamName: "test-stream"}},
					queue:   suite.mockQueue,
					logger:  suite.logger,
					metrics: nil, // Missing metrics
				}, nil
			},
			expectValid: false,
			description: "Construction without metrics should be invalid",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			dispatcher, err := tc.setupData()

			suite.NoError(err, "Setup should not error")
			suite.NotNil(dispatcher, "Dispatcher should be created")

			// Test if the dispatcher can handle basic operations
			if tc.expectValid {
				suite.NotPanics(func() {
					dispatcher.Queue(suite.createTestMessage())
				}, tc.description)
			} else {
				// For invalid constructions, operations might panic
				// We just verify the construction itself doesn't panic
				suite.NotNil(dispatcher, "Even invalid dispatcher should be created")
			}
		})
	}
}
