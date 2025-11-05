// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"errors"
	"testing"

	"github.com/xmidt-org/xmidt-event-streams/sender"

	"github.com/fogfish/opts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type DispatcherFactoryTestSuite struct {
	suite.Suite
	logger *zap.Logger
}

func TestDispatcherFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(DispatcherFactoryTestSuite))
}

func (suite *DispatcherFactoryTestSuite) SetupTest() {
	suite.logger = zaptest.NewLogger(suite.T())
}

func (suite *DispatcherFactoryTestSuite) createTestBaseFilter() *BaseFilter {
	return &BaseFilter{
		id:         "test-filter",
		queueSize:  100,
		batchSize:  10,
		maxWorkers: 5,
	}
}

func (suite *DispatcherFactoryTestSuite) createTestStreams() []Stream {
	return []Stream{
		{
			StreamName: "test-stream-1",
			Config: map[string]string{
				"region": "us-west-2",
				"role":   "test-role",
			},
		},
		{
			StreamName: "test-stream-2",
			Config: map[string]string{
				"region": "us-east-1",
				"role":   "test-role-2",
			},
		},
	}
}

func (suite *DispatcherFactoryTestSuite) TestNewDispatcherFactory() {
	suite.T().Run("successful_creation", func(t *testing.T) {
		mockSenderProvider := &MockSenderProvider{}
		mockQueueProvider := NewMockQueueProvider()

		// Create mock metrics
		mockCounter := NewMockCounter()
		mockCounter.On("With", mock.Anything).Return(mockCounter)
		mockCounter.On("Add", mock.Anything).Return()

		mockMetrics := &FilterMetrics{
			DroppedMessage: mockCounter,
		}

		factory, err := NewDispatcherFactory([]opts.Option[DispatcherFactory]{
			WithSenderProvider(mockSenderProvider),
			WithQueueProvider(mockQueueProvider),
			WithRetries(3),
			WithDispatchLogger(suite.logger),
			WithDispatchMetrics(mockMetrics),
		})
		suite.NoError(err)
		suite.NotNil(factory)
	})

	suite.T().Run("nil_sender_provider", func(t *testing.T) {
		mockQueueProvider := NewMockQueueProvider()

		// Create mock metrics
		mockCounter := NewMockCounter()
		mockCounter.On("With", mock.Anything).Return(mockCounter)
		mockCounter.On("Add", mock.Anything).Return()

		mockMetrics := &FilterMetrics{
			DroppedMessage: mockCounter,
		}

		_, err := NewDispatcherFactory([]opts.Option[DispatcherFactory]{
			WithSenderProvider(nil),
			WithQueueProvider(mockQueueProvider),
			WithRetries(3),
			WithDispatchLogger(suite.logger),
			WithDispatchMetrics(mockMetrics),
		})
		suite.Error(err)
		suite.Contains(err.Error(), "senderProvider cannot be nil")
	})

	suite.T().Run("negative_retries", func(t *testing.T) {
		mockSenderProvider := &MockSenderProvider{}
		mockQueueProvider := NewMockQueueProvider()

		// Create mock metrics
		mockCounter := NewMockCounter()
		mockCounter.On("With", mock.Anything).Return(mockCounter)
		mockCounter.On("Add", mock.Anything).Return()

		mockMetrics := &FilterMetrics{
			DroppedMessage: mockCounter,
		}

		_, err := NewDispatcherFactory([]opts.Option[DispatcherFactory]{
			WithSenderProvider(mockSenderProvider),
			WithQueueProvider(mockQueueProvider),
			WithRetries(-1),
			WithDispatchLogger(suite.logger),
			WithDispatchMetrics(mockMetrics),
		})
		suite.Error(err)
		suite.Contains(err.Error(), "retries cannot be negative")
	})
}

func (suite *DispatcherFactoryTestSuite) TestGetDispatcher_Success() {
	suite.T().Run("single_stream", func(t *testing.T) {
		// Create fresh mocks for this test
		mockSenderProvider := &MockSenderProvider{}
		mockQueueProvider := NewMockQueueProvider()
		mockSender1 := &MockSender{}
		mockQueue := NewMockQueue()

		// Create mock metrics
		mockCounter := NewMockCounter()
		mockCounter.On("With", mock.Anything).Return(mockCounter)
		mockCounter.On("Add", mock.Anything).Return()

		mockMetrics := &FilterMetrics{
			DroppedMessage: mockCounter,
		}

		factory, err := NewDispatcherFactory([]opts.Option[DispatcherFactory]{
			WithSenderProvider(mockSenderProvider),
			WithQueueProvider(mockQueueProvider),
			WithRetries(3),
			WithDispatchLogger(suite.logger),
			WithDispatchMetrics(mockMetrics),
		})
		suite.Require().NoError(err)

		filter := suite.createTestBaseFilter()
		streams := []Stream{suite.createTestStreams()[0]}
		destType := sender.Kinesis

		// Setup sender provider expectation
		mockSenderProvider.On("GetSender", 3, destType, streams[0].Config).
			Return(mockSender1, nil).Once()

		// Setup queue provider expectation
		mockQueueProvider.On("GetQueue",
			filter.id,
			filter.queueSize,
			filter.batchSize,
			filter.maxWorkers,
			mock.AnythingOfType("*filter.StreamDispatcher")).
			Return(mockQueue, nil).Once()

		// Note: queue.Start() is called in a goroutine, so we can't reliably test it
		// We'll just make sure it doesn't panic
		mockQueue.On("Start", mock.Anything).Return().Maybe()

		// Execute
		dispatcher, err := factory.GetDispatcher(filter, destType, streams)

		// Verify
		suite.NoError(err)
		suite.NotNil(dispatcher)
		suite.IsType(&StreamDispatcher{}, dispatcher)

		// Verify all expectations were met
		mockSenderProvider.AssertExpectations(suite.T())
		mockQueueProvider.AssertExpectations(suite.T())
		// Note: We don't assert queue expectations because Start() is called asynchronously
	})

	suite.T().Run("multiple_streams", func(t *testing.T) {
		// Create fresh mocks for this test
		mockSenderProvider := &MockSenderProvider{}
		mockQueueProvider := NewMockQueueProvider()
		mockSender1 := &MockSender{}
		mockSender2 := &MockSender{}
		mockQueue := NewMockQueue()

		// Create mock metrics
		mockCounter := NewMockCounter()
		mockCounter.On("With", mock.Anything).Return(mockCounter)
		mockCounter.On("Add", mock.Anything).Return()

		mockMetrics := &FilterMetrics{
			DroppedMessage: mockCounter,
		}

		factory, err := NewDispatcherFactory([]opts.Option[DispatcherFactory]{
			WithSenderProvider(mockSenderProvider),
			WithQueueProvider(mockQueueProvider),
			WithRetries(3),
			WithDispatchLogger(suite.logger),
			WithDispatchMetrics(mockMetrics),
		})
		suite.Require().NoError(err)

		filter := suite.createTestBaseFilter()
		streams := suite.createTestStreams()
		destType := sender.Kinesis

		// Setup sender provider expectations
		mockSenderProvider.On("GetSender", 3, destType, streams[0].Config).
			Return(mockSender1, nil).Once()
		mockSenderProvider.On("GetSender", 3, destType, streams[1].Config).
			Return(mockSender2, nil).Once()

		// Setup queue provider expectation
		mockQueueProvider.On("GetQueue",
			filter.id,
			filter.queueSize,
			filter.batchSize,
			filter.maxWorkers,
			mock.AnythingOfType("*filter.StreamDispatcher")).
			Return(mockQueue, nil).Once()

		// Note: queue.Start() is called in a goroutine, so we can't reliably test it
		mockQueue.On("Start", mock.Anything).Return().Maybe()

		// Execute
		dispatcher, err := factory.GetDispatcher(filter, destType, streams)

		// Verify
		suite.NoError(err)
		suite.NotNil(dispatcher)
		suite.IsType(&StreamDispatcher{}, dispatcher)

		// Type assert to check internal structure
		streamDispatcher, ok := dispatcher.(*StreamDispatcher)
		suite.True(ok)
		suite.Equal(len(streams), len(streamDispatcher.senders))
		suite.Equal(len(streams), len(streamDispatcher.streams))

		// Verify all expectations were met
		mockSenderProvider.AssertExpectations(suite.T())
		mockQueueProvider.AssertExpectations(suite.T())
	})

	suite.T().Run("empty_streams", func(t *testing.T) {
		// Create fresh mocks for this test
		mockSenderProvider := &MockSenderProvider{}
		mockQueueProvider := NewMockQueueProvider()
		mockQueue := NewMockQueue()

		// Create mock metrics
		mockCounter := NewMockCounter()
		mockCounter.On("With", mock.Anything).Return(mockCounter)
		mockCounter.On("Add", mock.Anything).Return()

		mockMetrics := &FilterMetrics{
			DroppedMessage: mockCounter,
		}

		factory, err := NewDispatcherFactory([]opts.Option[DispatcherFactory]{
			WithSenderProvider(mockSenderProvider),
			WithQueueProvider(mockQueueProvider),
			WithRetries(3),
			WithDispatchLogger(suite.logger),
			WithDispatchMetrics(mockMetrics),
		})
		suite.Require().NoError(err)

		filter := suite.createTestBaseFilter()
		streams := []Stream{}
		destType := sender.Kinesis

		// Setup queue provider expectation - no sender expectations needed for empty streams
		mockQueueProvider.On("GetQueue",
			filter.id,
			filter.queueSize,
			filter.batchSize,
			filter.maxWorkers,
			mock.AnythingOfType("*filter.StreamDispatcher")).
			Return(mockQueue, nil).Once()

		mockQueue.On("Start", mock.Anything).Return().Maybe()

		// Execute
		dispatcher, err := factory.GetDispatcher(filter, destType, streams)

		// Verify
		suite.NoError(err)
		suite.NotNil(dispatcher)
		suite.IsType(&StreamDispatcher{}, dispatcher)

		// Type assert to check internal structure
		streamDispatcher, ok := dispatcher.(*StreamDispatcher)
		suite.True(ok)
		suite.Equal(0, len(streamDispatcher.senders))
		suite.Equal(0, len(streamDispatcher.streams))

		// Verify all expectations were met
		mockQueueProvider.AssertExpectations(suite.T())
		// No sender provider expectations for empty streams
	})
}

func (suite *DispatcherFactoryTestSuite) TestGetDispatcher_SenderError() {
	suite.T().Run("sender_provider_error", func(t *testing.T) {
		// Create fresh mocks for this test
		mockSenderProvider := &MockSenderProvider{}
		mockQueueProvider := NewMockQueueProvider()

		// Create mock metrics
		mockCounter := NewMockCounter()
		mockCounter.On("With", mock.Anything).Return(mockCounter)
		mockCounter.On("Add", mock.Anything).Return()

		mockMetrics := &FilterMetrics{
			DroppedMessage: mockCounter,
		}

		factory, err := NewDispatcherFactory([]opts.Option[DispatcherFactory]{
			WithSenderProvider(mockSenderProvider),
			WithQueueProvider(mockQueueProvider),
			WithRetries(3),
			WithDispatchLogger(suite.logger),
			WithDispatchMetrics(mockMetrics),
		})
		suite.Require().NoError(err)

		filter := suite.createTestBaseFilter()
		streams := suite.createTestStreams()
		destType := sender.Kinesis

		// Setup sender provider to return error
		mockSenderProvider.On("GetSender", 3, destType, streams[0].Config).
			Return(nil, errors.New("sender creation failed")).Once()

		// Execute
		dispatcher, err := factory.GetDispatcher(filter, destType, streams)

		// Verify
		suite.Error(err)
		suite.Nil(dispatcher)
		suite.Contains(err.Error(), "error getting sender for dest type")
		suite.Contains(err.Error(), "sender creation failed")

		// Verify queue provider was not called
		mockQueueProvider.AssertNotCalled(suite.T(), "GetQueue")
		mockSenderProvider.AssertExpectations(suite.T())
	})
}

func (suite *DispatcherFactoryTestSuite) TestInterface_Compliance() {
	// Ensure DispatcherFactory implements DispatcherProvider interface
	var _ DispatcherProvider = (*DispatcherFactory)(nil)
}
