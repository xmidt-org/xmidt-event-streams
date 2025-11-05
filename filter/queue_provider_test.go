// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"github.com/xmidt-org/xmidt-event-streams/internal/queue"
	"testing"

	"github.com/fogfish/opts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockSubmitter implements the queue.Submitter interface for testing
type MockSubmitter struct {
	mock.Mock
}

func (m *MockSubmitter) Submit(items []*wrp.Message) error {
	args := m.Called(items)
	return args.Error(0)
}

// QueueProviderTestSuite contains tests for QueueProvider functionality
type QueueProviderTestSuite struct {
	suite.Suite
	logger        *zap.Logger
	telemetry     *queue.Telemetry
	mockSubmitter *MockSubmitter
}

func TestQueueProviderTestSuite(t *testing.T) {
	suite.Run(t, new(QueueProviderTestSuite))
}

func (suite *QueueProviderTestSuite) SetupTest() {
	// Create logger for testing
	suite.logger = zaptest.NewLogger(suite.T())

	// Create mock metrics for telemetry
	mockCounter := NewMockCounter()
	mockGauge := NewMockGauge()
	mockHistogram := NewMockHistogram()

	// Setup mock calls for metrics (these may be called during queue operations)
	mockCounter.On("With", mock.Anything).Return(mockCounter).Maybe()
	mockCounter.On("Add", mock.Anything).Return().Maybe()
	mockGauge.On("With", mock.Anything).Return(mockGauge).Maybe()
	mockGauge.On("Set", mock.Anything).Return().Maybe()
	mockGauge.On("Add", mock.Anything).Return().Maybe()
	mockHistogram.On("With", mock.Anything).Return(mockHistogram).Maybe()
	mockHistogram.On("Observe", mock.Anything).Return().Maybe()

	suite.telemetry = &queue.Telemetry{
		QueuedItems:  mockGauge,
		DroppedItems: mockCounter,
		BatchSize:    mockGauge,
		SubmitErrors: mockCounter,
		CallDuration: mockHistogram,
	}

	suite.mockSubmitter = &MockSubmitter{}
}

func (suite *QueueProviderTestSuite) TestNewQueueFactory_Success() {
	testCases := []struct {
		name        string
		options     []opts.Option[QueueFactory]
		expectError bool
	}{
		{
			name: "valid_configuration_with_all_options",
			options: []opts.Option[QueueFactory]{
				WithQueueTelemetry(suite.telemetry),
				WithQueueLogger(suite.logger),
			},
			expectError: false,
		},
		{
			name: "valid_configuration_minimal",
			options: []opts.Option[QueueFactory]{
				WithQueueTelemetry(suite.telemetry),
				WithQueueLogger(suite.logger),
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			factory, err := NewQueueFactory(tc.options)

			if tc.expectError {
				suite.Error(err)
				suite.Nil(factory)
			} else {
				suite.NoError(err)
				suite.NotNil(factory)
				suite.Implements((*QueueProvider)(nil), factory)
			}
		})
	}
}

func (suite *QueueProviderTestSuite) TestNewQueueFactory_ValidationErrors() {
	testCases := []struct {
		name          string
		options       []opts.Option[QueueFactory]
		expectedError string
	}{
		{
			name: "missing_telemetry",
			options: []opts.Option[QueueFactory]{
				WithQueueLogger(suite.logger),
			},
			expectedError: "Telemetry",
		},
		{
			name: "missing_logger",
			options: []opts.Option[QueueFactory]{
				WithQueueTelemetry(suite.telemetry),
			},
			expectedError: "Logger",
		},
		{
			name:          "missing_all_required",
			options:       []opts.Option[QueueFactory]{},
			expectedError: "Telemetry",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			factory, err := NewQueueFactory(tc.options)

			suite.Error(err)
			suite.Nil(factory)
			suite.Contains(err.Error(), tc.expectedError)
		})
	}
}

func (suite *QueueProviderTestSuite) TestQueueFactory_GetQueue_Success() {
	testCases := []struct {
		name        string
		queueName   string
		queueSize   int
		batchSize   int
		maxWorkers  int
		description string
	}{
		{
			name:        "standard_configuration",
			queueName:   "test-queue",
			queueSize:   100,
			batchSize:   10,
			maxWorkers:  5,
			description: "Standard queue configuration",
		},
	}

	// Create factory
	factory, err := NewQueueFactory([]opts.Option[QueueFactory]{
		WithQueueTelemetry(suite.telemetry),
		WithQueueLogger(suite.logger),
	})
	suite.NoError(err)
	suite.NotNil(factory)

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup mock submitter
			submitter := &MockSubmitter{}

			q, err := factory.GetQueue(tc.queueName, tc.queueSize, tc.batchSize, tc.maxWorkers, submitter)

			suite.NoError(err, "GetQueue should succeed for %s", tc.description)
			suite.NotNil(q, "Queue should not be nil for %s", tc.description)
			suite.Implements((*queue.Queue[*wrp.Message])(nil), q, "Queue should implement Queue interface")
		})
	}
}
