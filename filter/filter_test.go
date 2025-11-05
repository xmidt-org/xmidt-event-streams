// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// FilterTestSuite contains all the test cases for BaseFilter
type FilterTestSuite struct {
	suite.Suite
	logger         *zap.Logger
	mockDispatcher *MockDispatcher
}

func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}

func (suite *FilterTestSuite) SetupTest() {
	// Create logger for testing
	suite.logger = zaptest.NewLogger(suite.T())
	suite.mockDispatcher = new(MockDispatcher)
}

func (suite *FilterTestSuite) TearDownTest() {
	// Clean up any resources if needed
}

func (suite *FilterTestSuite) TestInterface() {
	// Test that BaseFilter implements the Filter interface
	var filter Filter = &BaseFilter{}
	suite.NotNil(filter, "BaseFilter should implement Filter interface")

	// Verify interface methods exist
	suite.Implements((*Filter)(nil), &BaseFilter{}, "BaseFilter should implement Filter interface")
}

func (suite *FilterTestSuite) TestOverlapsFunction() {
	testCases := []struct {
		name        string
		slice1      []string
		slice2      []string
		expected    bool
		description string
	}{
		{
			name:        "no_overlap",
			slice1:      []string{"a", "b", "c"},
			slice2:      []string{"d", "e", "f"},
			expected:    false,
			description: "Non-overlapping slices should return false",
		},
		{
			name:        "single_overlap",
			slice1:      []string{"a", "b", "c"},
			slice2:      []string{"c", "d", "e"},
			expected:    true,
			description: "Single overlapping element should return true",
		},
		{
			name:        "empty_slices",
			slice1:      []string{},
			slice2:      []string{},
			expected:    false,
			description: "Empty slices should return false",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result := overlaps(tc.slice1, tc.slice2)
			suite.Equal(tc.expected, result, tc.description)
		})
	}
}

func (suite *FilterTestSuite) TestShutdown() {
	testCases := []struct {
		name        string
		gentle      bool
		setupMocks  func()
		description string
	}{
		{
			name:   "gentle_shutdown",
			gentle: true,
			setupMocks: func() {
				suite.mockDispatcher.On("Shutdown", true).Return()
			},
			description: "Gentle shutdown should call dispatcher shutdown with gentle=true",
		},
		{
			name:   "abrupt_shutdown",
			gentle: false,
			setupMocks: func() {
				suite.mockDispatcher.On("Shutdown", false).Return()
			},
			description: "Abrupt shutdown should call dispatcher shutdown with gentle=false",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			filter := &BaseFilter{
				id:         "test-filter",
				logger:     suite.logger,
				dispatcher: suite.mockDispatcher,
			}

			tc.setupMocks()

			// Add a goroutine to the wait group to test wg.Done() is called
			filter.wg.Add(1)

			// Execute shutdown
			filter.Shutdown(tc.gentle)

			// Verify all expectations
			suite.mockDispatcher.AssertExpectations(suite.T())

			// Reset mocks for next test
			suite.mockDispatcher.ExpectedCalls = nil
		})
	}
}

func (suite *FilterTestSuite) TestQueue_EventMatching() {
	testCases := []struct {
		name          string
		eventMatchers []string
		message       *wrp.Message
		shouldQueue   bool
		setupMocks    func()
		description   string
	}{
		{
			name:          "event_matches_exact",
			eventMatchers: []string{"device-status"},
			message: &wrp.Message{
				Destination: "event:device-status",
				Source:      "device-123",
			},
			shouldQueue: true,
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Exact event match should queue message",
		},
		{
			name:          "event_does_not_match",
			eventMatchers: []string{"device-status"},
			message: &wrp.Message{
				Destination: "event:system-update",
				Source:      "device-789",
			},
			shouldQueue: false,
			setupMocks:  func() {},
			description: "Non-matching event should not queue message",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup event matchers
			eventMatchers := make([]*regexp.Regexp, len(tc.eventMatchers))
			for i, pattern := range tc.eventMatchers {
				eventMatchers[i] = regexp.MustCompile(pattern)
			}

			// Create filter with test configuration
			filter := &BaseFilter{
				id:               "test-filter",
				logger:           suite.logger,
				dispatcher:       suite.mockDispatcher,
				eventMatchers:    eventMatchers,
				metadataMatchers: nil,
			}

			tc.setupMocks()

			// Execute
			filter.Queue(tc.message)

			// Verify expectations
			if tc.shouldQueue {
				suite.mockDispatcher.AssertExpectations(suite.T())
			} else {
				suite.mockDispatcher.AssertNotCalled(suite.T(), "Queue")
			}

			// Reset mocks for next test
			suite.mockDispatcher.ExpectedCalls = nil
		})
	}
}

func (suite *FilterTestSuite) TestQueue_MetadataMatching() {
	testCases := []struct {
		name             string
		metadataMatchers []string
		message          *wrp.Message
		shouldQueue      bool
		setupMocks       func()
		description      string
	}{
		{
			name:             "metadata_matches_source",
			metadataMatchers: []string{"device-.*"},
			message: &wrp.Message{
				Destination: "event:device-status",
				Source:      "device-123",
			},
			shouldQueue: true,
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Metadata matcher matching source should queue message",
		},
		{
			name:             "metadata_does_not_match",
			metadataMatchers: []string{"allowed-.*"},
			message: &wrp.Message{
				Destination: "event:device-status",
				Source:      "blocked-device",
			},
			shouldQueue: false,
			setupMocks:  func() {},
			description: "Non-matching metadata should not queue message",
		},
		{
			name:             "nil_metadata_matchers",
			metadataMatchers: nil,
			message: &wrp.Message{
				Destination: "event:any-event",
				Source:      "any-source",
			},
			shouldQueue: true,
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Nil metadata matchers should allow all messages",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup event matcher (always allow events for metadata testing)
			eventMatchers := []*regexp.Regexp{regexp.MustCompile(".*")}

			// Setup metadata matchers
			var metadataMatchers []*regexp.Regexp
			if tc.metadataMatchers != nil {
				metadataMatchers = make([]*regexp.Regexp, len(tc.metadataMatchers))
				for i, pattern := range tc.metadataMatchers {
					metadataMatchers[i] = regexp.MustCompile(pattern)
				}
			}

			// Create filter with test configuration
			filter := &BaseFilter{
				id:               "test-filter",
				logger:           suite.logger,
				dispatcher:       suite.mockDispatcher,
				eventMatchers:    eventMatchers,
				metadataMatchers: metadataMatchers,
			}

			tc.setupMocks()

			// Execute
			filter.Queue(tc.message)

			// Verify expectations
			if tc.shouldQueue {
				suite.mockDispatcher.AssertExpectations(suite.T())
			} else {
				suite.mockDispatcher.AssertNotCalled(suite.T(), "Queue")
			}

			// Reset mocks for next test
			suite.mockDispatcher.ExpectedCalls = nil
		})
	}
}

func (suite *FilterTestSuite) TestQueue_CombinedEventAndMetadataMatching() {
	testCases := []struct {
		name             string
		eventMatchers    []string
		metadataMatchers []string
		message          *wrp.Message
		shouldQueue      bool
		setupMocks       func()
		description      string
	}{
		{
			name:             "both_event_and_metadata_match",
			eventMatchers:    []string{"device-.*"},
			metadataMatchers: []string{"device-.*"},
			message: &wrp.Message{
				Destination: "event:device-status",
				Source:      "device-123",
			},
			shouldQueue: true,
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Both event and metadata match should queue",
		},
		{
			name:             "event_matches_metadata_does_not",
			eventMatchers:    []string{"device-.*"},
			metadataMatchers: []string{"allowed-.*"},
			message: &wrp.Message{
				Destination: "event:device-status",
				Source:      "device-123",
			},
			shouldQueue: false,
			setupMocks:  func() {},
			description: "Event matches but metadata doesn't - should not queue",
		},
		{
			name:             "metadata_matches_event_does_not",
			eventMatchers:    []string{"allowed-.*"},
			metadataMatchers: []string{"device-.*"},
			message: &wrp.Message{
				Destination: "event:device-status",
				Source:      "device-123",
			},
			shouldQueue: false,
			setupMocks:  func() {},
			description: "Metadata matches but event doesn't - should not queue",
		},
		{
			name:             "neither_matches",
			eventMatchers:    []string{"allowed-.*"},
			metadataMatchers: []string{"allowed-.*"},
			message: &wrp.Message{
				Destination: "event:device-status",
				Source:      "device-123",
			},
			shouldQueue: false,
			setupMocks:  func() {},
			description: "Neither event nor metadata match - should not queue",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup event matchers
			eventMatchers := make([]*regexp.Regexp, len(tc.eventMatchers))
			for i, pattern := range tc.eventMatchers {
				eventMatchers[i] = regexp.MustCompile(pattern)
			}

			// Setup metadata matchers
			metadataMatchers := make([]*regexp.Regexp, len(tc.metadataMatchers))
			for i, pattern := range tc.metadataMatchers {
				metadataMatchers[i] = regexp.MustCompile(pattern)
			}

			// Create filter with test configuration
			filter := &BaseFilter{
				id:               "test-filter",
				logger:           suite.logger,
				dispatcher:       suite.mockDispatcher,
				eventMatchers:    eventMatchers,
				metadataMatchers: metadataMatchers,
			}

			tc.setupMocks()

			// Execute
			filter.Queue(tc.message)

			// Verify expectations
			if tc.shouldQueue {
				suite.mockDispatcher.AssertExpectations(suite.T())
			} else {
				suite.mockDispatcher.AssertNotCalled(suite.T(), "Queue")
			}

			// Reset mocks for next test
			suite.mockDispatcher.ExpectedCalls = nil
		})
	}
}

func (suite *FilterTestSuite) TestQueue_DestinationPrefixHandling() {
	testCases := []struct {
		name         string
		destination  string
		eventPattern string
		shouldQueue  bool
		setupMocks   func()
		description  string
	}{
		{
			name:         "strips_event_prefix",
			destination:  "event:device-status",
			eventPattern: "device-.*",
			shouldQueue:  true,
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Should strip 'event:' prefix before matching",
		},
		{
			name:         "no_prefix_to_strip",
			destination:  "device-status",
			eventPattern: "device-.*",
			shouldQueue:  true,
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Should handle destinations without 'event:' prefix",
		},
		{
			name:         "empty_destination",
			destination:  "",
			eventPattern: ".*",
			shouldQueue:  true, // ".*" matches empty string
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Empty destination should match .* pattern",
		},
		{
			name:         "event_prefix_only",
			destination:  "event:",
			eventPattern: ".*",
			shouldQueue:  true, // ".*" matches empty string after prefix removal
			setupMocks: func() {
				suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()
			},
			description: "Just 'event:' prefix with no event name should match .* pattern",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup matchers
			eventMatchers := []*regexp.Regexp{regexp.MustCompile(tc.eventPattern)}

			filter := &BaseFilter{
				id:               "test-filter",
				logger:           suite.logger,
				dispatcher:       suite.mockDispatcher,
				eventMatchers:    eventMatchers,
				metadataMatchers: nil, // No metadata filtering for this test
			}

			message := &wrp.Message{
				Destination: tc.destination,
				Source:      "test-source",
			}

			tc.setupMocks()

			// Execute
			filter.Queue(message)

			// Verify expectations
			if tc.shouldQueue {
				suite.mockDispatcher.AssertExpectations(suite.T())
			} else {
				suite.mockDispatcher.AssertNotCalled(suite.T(), "Queue")
			}

			// Reset mocks for next test
			suite.mockDispatcher.ExpectedCalls = nil
		})
	}
}

func (suite *FilterTestSuite) TestQueue_EdgeCases() {
	suite.Run("nil_message", func() {
		filter := &BaseFilter{
			id:               "test-filter",
			logger:           suite.logger,
			dispatcher:       suite.mockDispatcher,
			eventMatchers:    []*regexp.Regexp{regexp.MustCompile(".*")},
			metadataMatchers: nil,
		}

		// The filter doesn't handle nil messages gracefully - it will panic
		// This test documents the current behavior
		suite.Panics(func() {
			filter.Queue(nil)
		}, "Queue should panic with nil message")
	})

	suite.Run("complex_regex_patterns", func() {
		eventMatchers := []*regexp.Regexp{
			regexp.MustCompile("^device-(status|update|alert)$"),
		}
		metadataMatchers := []*regexp.Regexp{
			regexp.MustCompile("^(mac:|serial:).*"),
		}

		filter := &BaseFilter{
			id:               "test-filter",
			logger:           suite.logger,
			dispatcher:       suite.mockDispatcher,
			eventMatchers:    eventMatchers,
			metadataMatchers: metadataMatchers,
		}

		// Test matching complex patterns
		suite.mockDispatcher.On("Queue", mock.AnythingOfType("*wrp.Message")).Return()

		message := &wrp.Message{
			Destination: "event:device-status",
			Source:      "mac:AA:BB:CC:DD:EE:FF",
		}

		filter.Queue(message)
		suite.mockDispatcher.AssertExpectations(suite.T())

		// Reset for non-matching test
		suite.mockDispatcher.ExpectedCalls = nil

		// Test non-matching patterns
		message2 := &wrp.Message{
			Destination: "event:device-config", // doesn't match event pattern
			Source:      "mac:AA:BB:CC:DD:EE:FF",
		}

		filter.Queue(message2)
		suite.mockDispatcher.AssertNotCalled(suite.T(), "Queue")
	})
}
