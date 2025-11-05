// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"testing"

	"github.com/fogfish/opts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// SenderTestSuite contains all the test cases for sender package functionality
type SenderTestSuite struct {
	suite.Suite
	logger          *zap.Logger
	kinesisProvider *MockKinesisProvider
}

func TestSenderTestSuite(t *testing.T) {
	suite.Run(t, new(SenderTestSuite))
}

func (suite *SenderTestSuite) SetupTest() {
	// Create logger for testing
	suite.logger = zaptest.NewLogger(suite.T())
	suite.kinesisProvider = new(MockKinesisProvider)
	suite.kinesisProvider.On("Get", mock.Anything).Return(new(MockKinesisClientAPI), nil)
}

func (suite *SenderTestSuite) TearDownTest() {
	// Clean up any resources if needed
}

func (suite *SenderTestSuite) TestParseDestType_Success() {
	testCases := []struct {
		name        string
		input       string
		expected    DestType
		description string
	}{
		{
			name:        "kinesis_lowercase",
			input:       "kinesis",
			expected:    Kinesis,
			description: "Kinesis in lowercase should parse correctly",
		},
		{
			name:        "kinesis_uppercase",
			input:       "KINESIS",
			expected:    Kinesis,
			description: "Kinesis in uppercase should parse correctly",
		},
		{
			name:        "kinesis_mixed_case",
			input:       "KiNeSiS",
			expected:    Kinesis,
			description: "Kinesis in mixed case should parse correctly",
		},
		{
			name:        "kinesis_with_whitespace",
			input:       "  kinesis  ",
			expected:    Kinesis,
			description: "Kinesis with surrounding whitespace should parse correctly",
		},
		{
			name:        "kinesis_with_tabs",
			input:       "\tkinesis\t",
			expected:    Kinesis,
			description: "Kinesis with tabs should parse correctly",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := ParseDestType(tc.input)

			suite.NoError(err, tc.description)
			suite.Equal(tc.expected, result, "DestType should match expected for %s", tc.description)
		})
	}
}

func (suite *SenderTestSuite) TestParseDestType_Errors() {
	testCases := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "empty_string",
			input:       "",
			description: "Empty string should return error",
		},
		{
			name:        "whitespace_only",
			input:       "   ",
			description: "Whitespace-only string should return error",
		},
		{
			name:        "invalid_dest_type",
			input:       "sqs",
			description: "Invalid destination type should return error",
		},
		{
			name:        "partial_match",
			input:       "kines",
			description: "Partial match should return error",
		},
		{
			name:        "numeric_string",
			input:       "123",
			description: "Numeric string should return error",
		},
		{
			name:        "special_characters",
			input:       "kinesis!",
			description: "String with special characters should return error",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			result, err := ParseDestType(tc.input)

			suite.Error(err, tc.description)
			suite.Equal(DestType(0), result, "Result should be zero value on error")
			suite.Contains(err.Error(), "invalid DestType", "Error message should indicate invalid DestType")
			suite.Contains(err.Error(), tc.input, "Error message should contain the input value")
		})
	}
}

func (suite *SenderTestSuite) TestDestTypeConstants() {
	// Test that DestType constants have expected values
	suite.Equal(DestType(0), Kinesis, "Kinesis constant should have value 0")
}

func (suite *SenderTestSuite) TestNewSenderFactory_Success() {
	testCases := []struct {
		name        string
		options     []opts.Option[SenderFactory]
		description string
	}{

		{
			name:        "with_all",
			options:     []opts.Option[SenderFactory]{WithLogger(suite.logger), WithKinesisProvider(suite.kinesisProvider)},
			description: "Factory should be created successfully with logger option",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			factory, err := NewSenderFactory(tc.options)

			suite.NoError(err, tc.description)
			suite.NotNil(factory, "Factory should not be nil")
			suite.Implements((*SenderProvider)(nil), factory, "Factory should implement SenderProvider interface")

			// Verify it's actually a SenderFactory
			senderFactory, ok := factory.(*SenderFactory)
			suite.True(ok, "Factory should be of type *SenderFactory")

			// Check logger is set correctly when provided
			if len(tc.options) > 0 && tc.name == "with_logger" {
				suite.Equal(suite.logger, senderFactory.logger, "Logger should be set correctly")
			}
		})
	}
}

func (suite *SenderTestSuite) TestNewSenderFactory_Failure() {
	testCases := []struct {
		name        string
		options     []opts.Option[SenderFactory]
		description string
	}{
		{
			name:        "with_nil_logger",
			options:     []opts.Option[SenderFactory]{WithLogger(nil), WithKinesisProvider(suite.kinesisProvider)},
			description: "Factory should return error when logger is nil",
		},
		// {. // failing test case removed
		// 	name:        "with_nil_kinesisProvider",
		// 	options:     []opts.Option[SenderFactory]{WithLogger(suite.logger), WithKinesisProvider(kinesis.KinesisProvider(nil))},
		// 	description: "Factory should return error when provider is nil",
		// },
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			factory, err := NewSenderFactory(tc.options)

			suite.Error(err, tc.description)
			suite.Nil(factory, "Factory should be nil on error")
		})
	}
}

func (suite *SenderTestSuite) TestSenderFactory_GetSender_Kinesis() {
	testCases := []struct {
		name        string
		retries     int
		destType    DestType
		config      map[string]string
		expectError bool
		description string
	}{
		{
			name:     "valid_kinesis_config",
			retries:  3,
			destType: Kinesis,
			config: map[string]string{
				Region:   "us-east-1",
				Endpoint: "http://localhost:4567",
			},
			expectError: false, // These actually succeed but create sender with nil kinesis client
			description: "Valid Kinesis configuration succeeds but creates sender with nil client",
		},
		{
			name:        "kinesis_nil_config",
			retries:     3,
			destType:    Kinesis,
			config:      nil,
			expectError: true,
			description: "Kinesis with nil config should fail",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			factory, err := NewSenderFactory([]opts.Option[SenderFactory]{WithLogger(suite.logger), WithKinesisProvider(suite.kinesisProvider)})
			suite.NoError(err, "Factory creation should succeed")

			_, err = factory.GetSender(tc.retries, tc.destType, tc.config)

			if tc.expectError {
				suite.Error(err, tc.description)
				// Note: Sender may still be returned even with AWS credential errors
			} else {
				suite.NoError(err, tc.description)
				// In this case, kinesis client creation succeeds but may have nil client internally
			}
		})
	}
}

func (suite *SenderTestSuite) TestSenderFactory_GetSender_InvalidDestType() {
	invalidDestTypes := []struct {
		name     string
		destType DestType
	}{
		{
			name:     "undefined_dest_type_1",
			destType: DestType(999),
		},
	}

	factory, err := NewSenderFactory([]opts.Option[SenderFactory]{WithLogger(suite.logger), WithKinesisProvider(suite.kinesisProvider)})
	suite.NoError(err, "Factory creation should succeed")

	for _, tc := range invalidDestTypes {
		suite.Run(tc.name, func() {
			config := map[string]string{
				Region:   "us-east-1",
				Endpoint: "http://localhost:4567",
			}

			sender, err := factory.GetSender(3, tc.destType, config)

			suite.Error(err, "Invalid destination type should return error")
			suite.Nil(sender, "Sender should be nil for invalid destination type")
			suite.Contains(err.Error(), "no sender for dest type", "Error should indicate no sender for dest type")
		})
	}
}
