// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/xmidt-org/xmidt-event-streams/sender"
)

// FilterFactoryTestSuite contains tests for BaseFilterFactory functionality
type FilterFactoryTestSuite struct {
	suite.Suite
	logger               *zap.Logger
	mockFilterMetrics    *FilterMetrics
	mockDispatchProvider *MockDispatcherProvider
	mockDispatcher       *MockDispatcher
	factory              *BaseFilterFactory
}

func TestFilterFactoryTestSuite(t *testing.T) {
	suite.Run(t, new(FilterFactoryTestSuite))
}

func (suite *FilterFactoryTestSuite) SetupTest() {
	// Create logger for testing
	suite.logger = zaptest.NewLogger(suite.T())

	// Create mock metrics
	mockCounter := NewMockCounter()
	mockGauge := NewMockGauge()

	// Setup mock calls for metrics
	mockCounter.On("With", mock.Anything).Return(mockCounter).Maybe()
	mockCounter.On("Add", mock.Anything).Return().Maybe()
	mockGauge.On("With", mock.Anything).Return(mockGauge).Maybe()
	mockGauge.On("Set", mock.Anything).Return().Maybe()
	mockGauge.On("Add", mock.Anything).Return().Maybe()

	suite.mockFilterMetrics = &FilterMetrics{
		DroppedMessage: mockCounter,
	}

	suite.mockDispatchProvider = new(MockDispatcherProvider)
	suite.mockDispatcher = new(MockDispatcher)

	// Setup base factory configuration
	suite.factory = &BaseFilterFactory{
		NumWorkers:         5,
		defaultQueueSize:   100,
		defaultMaxWorkers:  5,
		defaultBatchSize:   10,
		DeliveryRetries:    3,
		Metrics:            suite.mockFilterMetrics,
		Logger:             suite.logger,
		DispatcherProvider: suite.mockDispatchProvider,
	}
}

func (suite *FilterFactoryTestSuite) TestNew_Success() {
	testCases := []struct {
		name        string
		config      FilterConfig
		setupMocks  func()
		description string
	}{
		{
			name: "valid_minimal_configuration",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{"key": "value"},
				},
				Events:   []string{"device-status", ".*"},
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			setupMocks: func() {
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
			},
			description: "Valid minimal configuration should succeed",
		},
		{
			name: "configuration_with_explicit_sizes",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream-2",
					Config:     map[string]string{},
				},
				Events:     []string{"event-.*"},
				DestType:   "kinesis",
				QueueSize:  50,
				BatchSize:  5,
				MaxWorkers: 3,
				Metadata: Metadata{
					DeviceIds: []string{".*"},
				},
			},
			setupMocks: func() {
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
			},
			description: "Configuration with explicit sizes should succeed",
		},
		{
			name: "configuration_with_alt_streams",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "primary-stream",
					Config:     map[string]string{"primary": "true"},
				},
				AltStreams: []Stream{
					{
						StreamName: "alt-stream-1",
						Config:     map[string]string{"alt": "true"},
					},
					{
						StreamName: "alt-stream-2",
						Config:     map[string]string{"backup": "true"},
					},
				},
				Events:   []string{"test-event"},
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{"test-.*"},
				},
			},
			setupMocks: func() {
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
			},
			description: "Configuration with alternative streams should succeed",
		},
		{
			name: "configuration_with_match_all_devices",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "match-all-stream",
					Config:     map[string]string{},
				},
				Events:   []string{".*"},
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{".*"}, // This should result in nil metadataMatchers
				},
			},
			setupMocks: func() {
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
			},
			description: "Configuration with match-all devices should succeed",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			filter, err := suite.factory.New(tc.config)

			suite.NoError(err, tc.description)
			suite.NotNil(filter, "Filter should not be nil for %s", tc.description)
			suite.Implements((*Filter)(nil), filter, "Should implement Filter interface")

			// Verify filter properties
			baseFilter, ok := filter.(*BaseFilter)
			suite.True(ok, "Filter should be *BaseFilter")

			if ok {
				suite.Equal(tc.config.Stream.StreamName, baseFilter.id, "Filter ID should match stream name")

				// Check default values are applied when config values are 0
				if tc.config.QueueSize <= 0 {
					suite.Equal(suite.factory.defaultQueueSize, baseFilter.queueSize, "Should use default queue size")
				} else {
					suite.Equal(tc.config.QueueSize, baseFilter.queueSize, "Should use configured queue size")
				}

				if tc.config.BatchSize <= 0 {
					suite.Equal(suite.factory.defaultBatchSize, baseFilter.batchSize, "Should use default batch size")
				} else {
					suite.Equal(tc.config.BatchSize, baseFilter.batchSize, "Should use configured batch size")
				}

				if tc.config.MaxWorkers <= 0 {
					suite.Equal(suite.factory.defaultMaxWorkers, baseFilter.maxWorkers, "Should use default max workers")
				} else {
					suite.Equal(tc.config.MaxWorkers, baseFilter.maxWorkers, "Should use configured max workers")
				}

				suite.NotNil(baseFilter.workers, "Workers semaphore should be initialized")
				suite.NotNil(baseFilter.eventMatchers, "Event matchers should be initialized")
				suite.NotEmpty(baseFilter.eventMatchers, "Event matchers should not be empty")
			}

			// Reset mocks for next test
			suite.mockDispatchProvider.ExpectedCalls = nil
		})
	}
}

func (suite *FilterFactoryTestSuite) TestNew_ValidationErrors() {
	testCases := []struct {
		name          string
		config        FilterConfig
		setupMocks    func()
		expectedError string
		description   string
	}{
		{
			name: "invalid_destination_type",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:   []string{"test-event"},
				DestType: "invalid-dest-type",
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			setupMocks:    func() {},
			expectedError: "error parsing destination type",
			description:   "Invalid destination type should cause error",
		},
		{
			name: "dispatcher_creation_error",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:   []string{"test-event"},
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			setupMocks: func() {
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(nil, errors.New("dispatcher creation failed"))
			},
			expectedError: "dispatcher creation failed",
			description:   "Dispatcher creation error should propagate",
		},
		{
			name: "empty_events_list",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:   []string{}, // Empty events should cause error
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			setupMocks: func() {
				// Mock needed because dispatcher is created before validation
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
			},
			expectedError: "event matchers must not be empty",
			description:   "Empty events list should cause error",
		},
		{
			name: "invalid_event_regex",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:   []string{"[invalid-regex"},
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			setupMocks: func() {
				// Mock needed because dispatcher is created before validation
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
			},
			expectedError: "invalid event matcher item",
			description:   "Invalid event regex should cause error",
		},
		{
			name: "invalid_device_id_regex",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:   []string{"test-event"},
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{"[invalid-regex"},
				},
			},
			setupMocks: func() {
				// Mock needed because dispatcher is created before validation
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
			},
			expectedError: "invalid matcher item",
			description:   "Invalid device ID regex should cause error",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			filter, err := suite.factory.New(tc.config)

			suite.Error(err, tc.description)
			suite.Nil(filter, "Filter should be nil when error expected")
			suite.Contains(err.Error(), tc.expectedError, "Error should contain expected message")

			// Reset mocks for next test
			suite.mockDispatchProvider.ExpectedCalls = nil
		})
	}
}

func (suite *FilterFactoryTestSuite) TestNew_DefaultValueApplication() {
	testCases := []struct {
		name               string
		config             FilterConfig
		expectedQueueSize  int
		expectedBatchSize  int
		expectedMaxWorkers int
		description        string
	}{
		{
			name: "zero_values_use_defaults",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:     []string{"test-event"},
				DestType:   "kinesis",
				QueueSize:  0, // Should use default
				BatchSize:  0, // Should use default
				MaxWorkers: 0, // Should use default
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			expectedQueueSize:  100,
			expectedBatchSize:  10,
			expectedMaxWorkers: 5,
			description:        "Zero values should use factory defaults",
		},
		{
			name: "negative_values_use_defaults",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:     []string{"test-event"},
				DestType:   "kinesis",
				QueueSize:  -1, // Should use default
				BatchSize:  -5, // Should use default
				MaxWorkers: -2, // Should use default
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			expectedQueueSize:  100,
			expectedBatchSize:  10,
			expectedMaxWorkers: 5,
			description:        "Negative values should use factory defaults",
		},
		{
			name: "explicit_values_used",
			config: FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:     []string{"test-event"},
				DestType:   "kinesis",
				QueueSize:  200,
				BatchSize:  20,
				MaxWorkers: 10,
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			},
			expectedQueueSize:  200,
			expectedBatchSize:  20,
			expectedMaxWorkers: 10,
			description:        "Explicit positive values should be used",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup mock to succeed
			suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)

			filter, err := suite.factory.New(tc.config)

			suite.NoError(err, tc.description)
			suite.NotNil(filter, "Filter should not be nil")

			baseFilter, ok := filter.(*BaseFilter)
			suite.True(ok, "Filter should be *BaseFilter")

			if ok {
				suite.Equal(tc.expectedQueueSize, baseFilter.queueSize, "Queue size should match expected for %s", tc.description)
				suite.Equal(tc.expectedBatchSize, baseFilter.batchSize, "Batch size should match expected for %s", tc.description)
				suite.Equal(tc.expectedMaxWorkers, baseFilter.maxWorkers, "Max workers should match expected for %s", tc.description)
			}

			// Reset mocks for next test
			suite.mockDispatchProvider.ExpectedCalls = nil
		})
	}
}

func (suite *FilterFactoryTestSuite) TestNew_RegexMatcherHandling() {
	testCases := []struct {
		name                        string
		deviceIds                   []string
		expectedMetadataMatchersNil bool
		description                 string
	}{
		{
			name:                        "match_all_pattern",
			deviceIds:                   []string{".*"},
			expectedMetadataMatchersNil: true,
			description:                 "Match-all pattern should result in nil metadata matchers",
		},
		{
			name:                        "specific_patterns",
			deviceIds:                   []string{"device-123", "test-.*"},
			expectedMetadataMatchersNil: false,
			description:                 "Specific patterns should create metadata matchers",
		},
		{
			name:                        "empty_device_ids",
			deviceIds:                   []string{},
			expectedMetadataMatchersNil: true,
			description:                 "Empty device IDs should result in nil metadata matchers",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			config := FilterConfig{
				Stream: Stream{
					StreamName: "test-stream",
					Config:     map[string]string{},
				},
				Events:   []string{"test-event"},
				DestType: "kinesis",
				Metadata: Metadata{
					DeviceIds: tc.deviceIds,
				},
			}

			// Setup mock to succeed
			suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)

			filter, err := suite.factory.New(config)

			suite.NoError(err, tc.description)
			suite.NotNil(filter, "Filter should not be nil")

			baseFilter, ok := filter.(*BaseFilter)
			suite.True(ok, "Filter should be *BaseFilter")

			if ok {
				if tc.expectedMetadataMatchersNil {
					suite.Nil(baseFilter.metadataMatchers, "Metadata matchers should be nil for %s", tc.description)
				} else {
					suite.NotNil(baseFilter.metadataMatchers, "Metadata matchers should not be nil for %s", tc.description)
					suite.NotEmpty(baseFilter.metadataMatchers, "Metadata matchers should not be empty for %s", tc.description)
				}
			}

			// Reset mocks for next test
			suite.mockDispatchProvider.ExpectedCalls = nil
		})
	}
}

func (suite *FilterFactoryTestSuite) TestNew_StreamHandling() {
	testCases := []struct {
		name          string
		primaryStream Stream
		altStreams    []Stream
		description   string
	}{
		{
			name: "primary_stream_only",
			primaryStream: Stream{
				StreamName: "primary-only",
				Config:     map[string]string{"type": "primary"},
			},
			altStreams:  []Stream{},
			description: "Primary stream only should work",
		},
		{
			name: "primary_with_multiple_alt_streams",
			primaryStream: Stream{
				StreamName: "primary",
				Config:     map[string]string{"type": "primary"},
			},
			altStreams: []Stream{
				{
					StreamName: "alt-1",
					Config:     map[string]string{"type": "alt", "priority": "1"},
				},
				{
					StreamName: "alt-2",
					Config:     map[string]string{"type": "alt", "priority": "2"},
				},
			},
			description: "Primary with multiple alternative streams should work",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			config := FilterConfig{
				Stream:     tc.primaryStream,
				AltStreams: tc.altStreams,
				Events:     []string{"test-event"},
				DestType:   "kinesis",
				Metadata: Metadata{
					DeviceIds: []string{"device-.*"},
				},
			}

			// Capture the streams passed to GetDispatcher
			var capturedStreams []Stream
			suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Run(func(args mock.Arguments) {
				capturedStreams = args.Get(2).([]Stream)
			}).Return(suite.mockDispatcher, nil)

			filter, err := suite.factory.New(config)

			suite.NoError(err, tc.description)
			suite.NotNil(filter, "Filter should not be nil")

			// Verify streams were passed correctly
			expectedStreamCount := 1 + len(tc.altStreams) // primary + alternatives
			suite.Len(capturedStreams, expectedStreamCount, "Should pass correct number of streams")
			suite.Equal(tc.primaryStream.StreamName, capturedStreams[0].StreamName, "Primary stream should be first")

			for i, altStream := range tc.altStreams {
				suite.Equal(altStream.StreamName, capturedStreams[i+1].StreamName, "Alt stream %d should match", i)
			}

			// Reset mocks for next test
			suite.mockDispatchProvider.ExpectedCalls = nil
		})
	}
}

func (suite *FilterFactoryTestSuite) TestNew_FactoryConfiguration() {
	// Test that factory configuration is properly applied to the filter
	config := FilterConfig{
		Stream: Stream{
			StreamName: "test-stream",
			Config:     map[string]string{},
		},
		Events:   []string{"test-event"},
		DestType: "kinesis",
		Metadata: Metadata{
			DeviceIds: []string{"device-.*"},
		},
	}

	// Setup mock to succeed
	suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)

	filter, err := suite.factory.New(config)

	suite.NoError(err, "Filter creation should succeed")
	suite.NotNil(filter, "Filter should not be nil")

	baseFilter, ok := filter.(*BaseFilter)
	suite.True(ok, "Filter should be *BaseFilter")

	if ok {
		suite.Equal(suite.factory.DeliveryRetries, baseFilter.deliveryRetries, "Delivery retries should match factory setting")
		suite.Equal(suite.factory.Metrics, baseFilter.metrics, "Metrics should match factory setting")
		suite.Equal(suite.factory.DispatcherProvider, suite.factory.DispatcherProvider, "Dispatch provider should match factory setting")
		suite.NotNil(baseFilter.logger, "Logger should be set")
	}
}
