// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fogfish/opts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"eventstream/sender"
)

// FilterManagerTestSuite contains tests for FilterManager functionality
type FilterManagerTestSuite struct {
	suite.Suite
	logger               *zap.Logger
	mockFilterMetrics    *FilterMetrics
	mockManagerMetrics   *FilterManagerMetrics
	mockDispatchProvider *MockDispatcherProvider
	mockDispatcher       *MockDispatcher
	filterManager        *BaseFilterManager
}

func TestFilterManagerTestSuite(t *testing.T) {
	suite.Run(t, new(FilterManagerTestSuite))
}

func (suite *FilterManagerTestSuite) SetupTest() {
	// Create logger for testing
	suite.logger = zaptest.NewLogger(suite.T())

	// Create mock metrics
	mockCounter := NewMockCounter()
	mockGauge := NewMockGauge()

	// Setup mock calls for metrics (these will be called during normal operations)
	mockCounter.On("With", mock.Anything).Return(mockCounter).Maybe()
	mockCounter.On("Add", mock.Anything).Return().Maybe()
	mockGauge.On("With", mock.Anything).Return(mockGauge).Maybe()
	mockGauge.On("Set", mock.Anything).Return().Maybe()
	mockGauge.On("Add", mock.Anything).Return().Maybe()

	suite.mockFilterMetrics = &FilterMetrics{
		DroppedMessage: mockCounter,
	}

	suite.mockManagerMetrics = &FilterManagerMetrics{
		EventType: mockCounter,
	}

	suite.mockDispatchProvider = new(MockDispatcherProvider)
	suite.mockDispatcher = new(MockDispatcher)
	filterManager, err := New([]opts.Option[BaseFilterManager]{
		WithLogger(suite.logger),
		WithFilterMetrics(suite.mockFilterMetrics),
		WithFilterManagerMetrics(suite.mockManagerMetrics),
		WithDispatchProvider(suite.mockDispatchProvider),
		WithFilters([]FilterConfig{}),
		WithDefaultQueueSize(100),
		WithDefaultBatchSize(10),
		WithDefaultWorkers(5),
	})
	suite.NoError(err)
	suite.filterManager = filterManager.(*BaseFilterManager)
}

func (suite *FilterManagerTestSuite) TestNew_Success() {
	testCases := []struct {
		name        string
		options     []opts.Option[BaseFilterManager]
		expectError bool
		description string
	}{
		{
			name: "minimal_valid_configuration",
			options: []opts.Option[BaseFilterManager]{
				WithLogger(suite.logger),
				WithFilterMetrics(suite.mockFilterMetrics),
				WithFilterManagerMetrics(suite.mockManagerMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithFilters([]FilterConfig{}),
				WithDefaultQueueSize(100),
				WithDefaultBatchSize(10),
				WithDefaultWorkers(5),
			},
			expectError: false,
			description: "Minimal valid configuration should succeed",
		},
		{
			name: "full_configuration",
			options: []opts.Option[BaseFilterManager]{
				WithLogger(suite.logger),
				WithFilterMetrics(suite.mockFilterMetrics),
				WithFilterManagerMetrics(suite.mockManagerMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithFilters([]FilterConfig{
					{
						Stream: Stream{
							StreamName: "test-stream",
							Config:     map[string]string{"key": "value"},
						},
						Events: []string{"event1"},
						Metadata: Metadata{
							DeviceIds: []string{"device-.*"},
						},
						DestType:   "kinesis",
						QueueSize:  50,
						BatchSize:  5,
						MaxWorkers: 3,
					},
				}),
				WithDefaultQueueSize(100),
				WithDefaultBatchSize(10),
				WithDefaultWorkers(5),
				WithDeliveryRetries(3),
			},
			expectError: false,
			description: "Full configuration should succeed",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup mocks for filter creation if needed
			if len(tc.options) > 5 { // Has filters
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil).Maybe()
				suite.mockDispatcher.On("Shutdown", mock.AnythingOfType("bool")).Return().Maybe()
			}

			manager, err := New(tc.options)

			if tc.expectError {
				suite.Error(err, tc.description)
				suite.Nil(manager, "Manager should be nil when error expected")
			} else {
				suite.NoError(err, tc.description)
				suite.NotNil(manager, "Manager should not be nil when no error expected")
				suite.Implements((*FilterManager)(nil), manager, "Should implement FilterManager interface")

				// Clean up
				manager.Shutdown(false)
			}

			// Reset mocks for next test
			suite.mockDispatchProvider.ExpectedCalls = nil
			suite.mockDispatcher.ExpectedCalls = nil
		})
	}
}

func (suite *FilterManagerTestSuite) TestNew_ValidationErrors() {
	testCases := []struct {
		name          string
		options       []opts.Option[BaseFilterManager]
		expectedError string
	}{
		{
			name: "missing_logger",
			options: []opts.Option[BaseFilterManager]{
				WithFilterMetrics(suite.mockFilterMetrics),
				WithFilterManagerMetrics(suite.mockManagerMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithFilters([]FilterConfig{}),
				WithDefaultQueueSize(100),
				WithDefaultBatchSize(10),
				WithDefaultWorkers(5),
			},
			expectedError: "Logger",
		},
		{
			name: "missing_filter_metrics",
			options: []opts.Option[BaseFilterManager]{
				WithLogger(suite.logger),
				WithFilterManagerMetrics(suite.mockManagerMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithFilters([]FilterConfig{}),
				WithDefaultQueueSize(100),
				WithDefaultBatchSize(10),
				WithDefaultWorkers(5),
			},
			expectedError: "FilterMetrics",
		},
		{
			name: "missing_manager_metrics",
			options: []opts.Option[BaseFilterManager]{
				WithLogger(suite.logger),
				WithFilterMetrics(suite.mockFilterMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithFilters([]FilterConfig{}),
				WithDefaultQueueSize(100),
				WithDefaultBatchSize(10),
				WithDefaultWorkers(5),
			},
			expectedError: "FilterManagerMetrics",
		},
		{
			name: "missing_filters",
			options: []opts.Option[BaseFilterManager]{
				WithLogger(suite.logger),
				WithFilterMetrics(suite.mockFilterMetrics),
				WithFilterManagerMetrics(suite.mockManagerMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithDefaultQueueSize(100),
				WithDefaultBatchSize(10),
				WithDefaultWorkers(5),
			},
			expectedError: "[]filter.FilterConfig",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			manager, err := New(tc.options)

			suite.Error(err)
			suite.Contains(err.Error(), tc.expectedError)

			// Manager might still be returned even with validation errors
			if manager != nil {
				manager.Shutdown(false)
			}
		})
	}
}

func (suite *FilterManagerTestSuite) TestLoadFilters() {
	testCases := []struct {
		name               string
		filterConfigs      []FilterConfig
		setupMocks         func()
		expectError        bool
		expectedFiltersLen int
		description        string
	}{
		{
			name:               "load_no_filters",
			filterConfigs:      []FilterConfig{},
			setupMocks:         func() {},
			expectError:        false,
			expectedFiltersLen: 0,
			description:        "Loading no filters should succeed",
		},
		{
			name: "load_valid_filters",
			filterConfigs: []FilterConfig{
				{
					Stream: Stream{
						StreamName: "test-stream-1",
						Config:     map[string]string{"key": "value"},
					},
					Events:   []string{"event-.*"},
					DestType: "kinesis",
					Metadata: Metadata{
						DeviceIds: []string{"device-.*"},
					},
				},
				{
					Stream: Stream{
						StreamName: "test-stream-2",
						Config:     map[string]string{},
					},
					Events:   []string{"test-event"},
					DestType: "kinesis",
					Metadata: Metadata{
						DeviceIds: []string{".*"},
					},
				},
			},
			setupMocks: func() {
				suite.mockDispatchProvider.On("GetDispatcher", mock.AnythingOfType("*filter.BaseFilter"), sender.Kinesis, mock.AnythingOfType("[]filter.Stream")).Return(suite.mockDispatcher, nil)
				suite.mockDispatcher.On("Shutdown", mock.AnythingOfType("bool")).Return()
			},
			expectError:        false,
			expectedFiltersLen: 2,
			description:        "Loading valid filters should succeed",
		},
		{
			name: "load_filters_with_creation_errors",
			filterConfigs: []FilterConfig{
				{
					Stream: Stream{
						StreamName: "test-stream",
					},
					DestType: "", // Invalid empty dest type will cause error
				},
			},
			setupMocks: func() {
				// No mock setup needed - should fail before dispatcher creation
			},
			expectError:        true,
			expectedFiltersLen: 0,
			description:        "Filter creation errors should propagate",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			manager, err := New([]opts.Option[BaseFilterManager]{
				WithLogger(suite.logger),
				WithFilterMetrics(suite.mockFilterMetrics),
				WithFilterManagerMetrics(suite.mockManagerMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithFilters(tc.filterConfigs),
				WithDefaultQueueSize(100),
				WithDefaultBatchSize(10),
				WithDefaultWorkers(5),
			})

			if tc.expectError {
				suite.Error(err, tc.description)
			} else {
				suite.NoError(err, tc.description)
				suite.NotNil(manager, "Manager should not be nil")

				// Verify filters were loaded
				baseManager := manager.(*BaseFilterManager)
				suite.Len(baseManager.filters, tc.expectedFiltersLen, "Should have expected number of filters")

				// Clean up
				manager.Shutdown(false)
			}

			// Reset mocks for next test
			suite.mockDispatchProvider.ExpectedCalls = nil
			suite.mockDispatcher.ExpectedCalls = nil
		})
	}
}

func (suite *FilterManagerTestSuite) TestQueue() {
	testCases := []struct {
		name          string
		setupManager  func() FilterManager
		message       *wrp.Message
		expectFilters int
		description   string
	}{
		{
			name: "queue_message_to_multiple_filters",
			setupManager: func() FilterManager {
				// Create mock filters
				mockFilter1 := new(MockFilter)
				mockFilter2 := new(MockFilter)

				mockFilter1.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
				mockFilter2.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()

				manager := suite.filterManager
				manager.filters = map[string]Filter{
					"filter1": mockFilter1,
					"filter2": mockFilter2,
				}
				return manager
			},
			message: &wrp.Message{
				Type:        wrp.SimpleEventMessageType,
				Source:      "device-123",
				Destination: "event:test-event",
			},
			expectFilters: 2,
			description:   "Message should be queued to all filters",
		},
		{
			name: "queue_message_with_no_filters",
			setupManager: func() FilterManager {
				manager := suite.filterManager
				manager.filters = map[string]Filter{}
				return manager
			},
			message: &wrp.Message{
				Type:        wrp.SimpleEventMessageType,
				Source:      "device-456",
				Destination: "event:another-event",
			},
			expectFilters: 0,
			description:   "Should handle case with no filters gracefully",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			manager := tc.setupManager()

			// This should not panic
			suite.NotPanics(func() {
				manager.Queue(tc.message)
			}, tc.description)

			// Verify that all mock filters were called appropriately
			if baseManager, ok := manager.(*BaseFilterManager); ok {
				for _, filter := range baseManager.filters {
					if mockFilter, ok := filter.(*MockFilter); ok {
						mockFilter.AssertCalled(suite.T(), "Queue", tc.message)
					}
				}
			}
		})
	}
}

func (suite *FilterManagerTestSuite) TestShutdown() {
	testCases := []struct {
		name        string
		gentle      bool
		numFilters  int
		description string
	}{
		{
			name:        "gentle_shutdown",
			gentle:      true,
			numFilters:  3,
			description: "Gentle shutdown should call Shutdown(true) on all filters",
		},
		{
			name:        "forceful_shutdown",
			gentle:      false,
			numFilters:  2,
			description: "Forceful shutdown should call Shutdown(false) on all filters",
		},
		{
			name:        "shutdown_with_no_filters",
			gentle:      true,
			numFilters:  0,
			description: "Shutdown with no filters should not panic",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Create mock filters
			mockFilters := make(map[string]Filter)
			for i := 0; i < tc.numFilters; i++ {
				mockFilter := new(MockFilter)
				mockFilter.On("Shutdown", tc.gentle).Return()
				mockFilters[fmt.Sprintf("filter%d", i)] = mockFilter
			}

			manager := &BaseFilterManager{
				filters:  mockFilters,
				mutex:    sync.RWMutex{},
				shutdown: make(chan struct{}),
			}

			// Shutdown should not panic
			suite.NotPanics(func() {
				manager.Shutdown(tc.gentle)
			}, tc.description)

			// Verify all filters were shut down
			for _, filter := range mockFilters {
				if mockFilter, ok := filter.(*MockFilter); ok {
					mockFilter.AssertCalled(suite.T(), "Shutdown", tc.gentle)
				}
			}

			// Verify filters map is cleared
			suite.Empty(manager.filters, "Filters map should be empty after shutdown")

			// Verify shutdown channel is closed
			select {
			case <-manager.shutdown:
				// Channel is closed, which is expected
			case <-time.After(100 * time.Millisecond):
				suite.Fail("Shutdown channel should be closed")
			}
		})
	}
}

func (suite *FilterManagerTestSuite) TestConcurrentAccess() {
	// Create a manager with some filters
	mockFilter1 := new(MockFilter)
	mockFilter2 := new(MockFilter)

	mockFilter1.On("Queue", mock.AnythingOfType("*wrp.Message")).Return().Maybe()
	mockFilter2.On("Queue", mock.AnythingOfType("*wrp.Message")).Return().Maybe()
	mockFilter1.On("Shutdown", mock.AnythingOfType("bool")).Return().Maybe()
	mockFilter2.On("Shutdown", mock.AnythingOfType("bool")).Return().Maybe()

	manager := suite.filterManager
	manager.filters = map[string]Filter{
		"filter1": mockFilter1,
		"filter2": mockFilter2,
	}

	// Test concurrent Queue operations
	const numGoroutines = 10
	const numMessages = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				msg := &wrp.Message{
					Type:        wrp.SimpleEventMessageType,
					Source:      fmt.Sprintf("device-%d-%d", goroutineID, j),
					Destination: "event:test-event",
				}
				manager.Queue(msg)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Now test shutdown while other operations might be happening
	suite.NotPanics(func() {
		manager.Shutdown(false)
	}, "Concurrent shutdown should not panic")
}

func (suite *FilterManagerTestSuite) TestCheckRequired() {
	testCases := []struct {
		name           string
		setupManager   func() *BaseFilterManager
		expectError    bool
		expectedFields []string
		description    string
	}{
		{
			name: "all_required_fields_present",
			setupManager: func() *BaseFilterManager {
				return &BaseFilterManager{
					logger:            suite.logger,
					filterMetrics:     suite.mockFilterMetrics,
					metrics:           suite.mockManagerMetrics,
					dispatchProvider:  suite.mockDispatchProvider,
					filterConfig:      []FilterConfig{},
					defaultQueueSize:  100,
					defaultBatchSize:  10,
					defaultMaxWorkers: 5,
				}
			},
			expectError:    false,
			expectedFields: []string{},
			description:    "Should pass validation with all required fields",
		},
		{
			name: "missing_logger",
			setupManager: func() *BaseFilterManager {
				return &BaseFilterManager{
					filterMetrics:     suite.mockFilterMetrics,
					metrics:           suite.mockManagerMetrics,
					dispatchProvider:  suite.mockDispatchProvider,
					filterConfig:      []FilterConfig{},
					defaultQueueSize:  100,
					defaultBatchSize:  10,
					defaultMaxWorkers: 5,
				}
			},
			expectError:    true,
			expectedFields: []string{"Logger"},
			description:    "Should fail validation with missing logger",
		},
		{
			name: "missing_filter_metrics",
			setupManager: func() *BaseFilterManager {
				return &BaseFilterManager{
					logger:            suite.logger,
					metrics:           suite.mockManagerMetrics,
					dispatchProvider:  suite.mockDispatchProvider,
					filterConfig:      []FilterConfig{},
					defaultQueueSize:  100,
					defaultBatchSize:  10,
					defaultMaxWorkers: 5,
				}
			},
			expectError:    true,
			expectedFields: []string{"FilterMetrics"},
			description:    "Should fail validation with missing filter metrics",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			manager := tc.setupManager()
			err := manager.checkRequired()

			if tc.expectError {
				suite.Error(err, tc.description)
				for _, field := range tc.expectedFields {
					suite.Contains(err.Error(), field, "Error should mention missing field: %s", field)
				}
			} else {
				suite.NoError(err, tc.description)
			}
		})
	}
}

func (suite *FilterManagerTestSuite) TestDefaultValueConfiguration() {
	testCases := []struct {
		name               string
		queueSize          int
		batchSize          int
		maxWorkers         int
		expectedQueueSize  int
		expectedBatchSize  int
		expectedMaxWorkers int
		description        string
	}{
		{
			name:               "explicit_values",
			queueSize:          200,
			batchSize:          20,
			maxWorkers:         10,
			expectedQueueSize:  200,
			expectedBatchSize:  20,
			expectedMaxWorkers: 10,
			description:        "Explicit values should be used",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			manager, err := New([]opts.Option[BaseFilterManager]{
				WithLogger(suite.logger),
				WithFilterMetrics(suite.mockFilterMetrics),
				WithFilterManagerMetrics(suite.mockManagerMetrics),
				WithDispatchProvider(suite.mockDispatchProvider),
				WithFilters([]FilterConfig{}),
				WithDefaultQueueSize(tc.queueSize),
				WithDefaultBatchSize(tc.batchSize),
				WithDefaultWorkers(tc.maxWorkers),
			})

			suite.NoError(err, tc.description)
			suite.NotNil(manager, "Manager should not be nil")

			baseManager := manager.(*BaseFilterManager)
			suite.Equal(tc.expectedQueueSize, baseManager.defaultQueueSize, "Queue size should match expected for %s", tc.description)
			suite.Equal(tc.expectedBatchSize, baseManager.defaultBatchSize, "Batch size should match expected for %s", tc.description)
			suite.Equal(tc.expectedMaxWorkers, baseManager.defaultMaxWorkers, "Max workers should match expected for %s", tc.description)

			// Clean up
			manager.Shutdown(false)
		})
	}
}
