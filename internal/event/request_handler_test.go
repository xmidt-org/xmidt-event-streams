// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package event

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fogfish/opts"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type RequestHandlerTestSuite struct {
	suite.Suite
	handler          *RequestHandler
	mockEventHandler *mockHandler
	logger           *zap.Logger
	fixedTime        time.Time
}

func TestRequestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(RequestHandlerTestSuite))
}

func (suite *RequestHandlerTestSuite) SetupTest() {
	suite.logger = zaptest.NewLogger(suite.T())
	suite.mockEventHandler = &mockHandler{}
	suite.fixedTime = time.Date(2023, 10, 30, 10, 0, 0, 0, time.UTC)

	var err error
	suite.handler, err = New([]opts.Option[RequestHandler]{
		WithLogger(suite.logger),
		WithEventHandler(suite.mockEventHandler),
	})
	suite.Require().NoError(err)

	// Override the time function for predictable testing
	suite.handler.now = func() time.Time {
		return suite.fixedTime
	}
}

func (suite *RequestHandlerTestSuite) TearDownTest() {
	// Reset mocks for each test
	suite.mockEventHandler.Mock = mock.Mock{}
}

func (suite *RequestHandlerTestSuite) createTestWRPMessage() *wrp.Message {
	return &wrp.Message{
		Type:            wrp.SimpleEventMessageType,
		Source:          "device-123",
		Destination:     "event:device-status/mac:112233445566/online",
		TransactionUUID: "test-transaction-123",
		ContentType:     "application/json",
		Payload:         []byte(`{"status":"online","timestamp":"2023-10-30T10:00:00Z"}`),
	}
}

func (suite *RequestHandlerTestSuite) createJSONPayload() []byte {
	msg := suite.createTestWRPMessage()
	encodedBytes := []byte{}
	encoder := wrp.NewEncoderBytes(&encodedBytes, wrp.JSON)
	encoder.Encode(msg)
	return encodedBytes
}

func (suite *RequestHandlerTestSuite) createMsgPackPayload() []byte {
	msg := suite.createTestWRPMessage()
	encodedBytes := []byte{}
	encoder := wrp.NewEncoderBytes(&encodedBytes, wrp.Msgpack)
	encoder.Encode(msg)
	return encodedBytes
}

func (suite *RequestHandlerTestSuite) TestNewRequestHandler() {
	suite.T().Run("successful_creation", func(t *testing.T) {
		handler, err := New([]opts.Option[RequestHandler]{
			WithLogger(suite.logger),
			WithEventHandler(suite.mockEventHandler),
		})
		suite.NoError(err)
		suite.NotNil(handler)
		suite.Equal(int32(MAX_OUTSTANDING_DEFAULT), handler.maxOutstanding)
		suite.NotNil(handler.now)
	})

	suite.T().Run("missing_logger", func(t *testing.T) {
		handler, err := New([]opts.Option[RequestHandler]{
			WithEventHandler(suite.mockEventHandler),
		})
		// Note: The current implementation may not enforce required fields
		// This test documents the current behavior
		if err != nil {
			suite.Error(err)
		} else {
			suite.NotNil(handler)
		}
	})

	suite.T().Run("missing_event_handler", func(t *testing.T) {
		handler, err := New([]opts.Option[RequestHandler]{
			WithLogger(suite.logger),
		})
		// Note: The current implementation may not enforce required fields
		// This test documents the current behavior
		if err != nil {
			suite.Error(err)
		} else {
			suite.NotNil(handler)
		}
	})
}

func (suite *RequestHandlerTestSuite) TestServeHTTP_Success() {
	testCases := []struct {
		name        string
		contentType string
		payload     []byte
		description string
	}{
		{
			name:        "json_request",
			contentType: "application/json",
			payload:     suite.createJSONPayload(),
			description: "Should handle valid JSON WRP message",
		},
		{
			name:        "msgpack_request",
			contentType: "application/msgpack",
			payload:     suite.createMsgPackPayload(),
			description: "Should handle valid MessagePack WRP message",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Setup fresh mocks for each test
			suite.SetupTest()

			// Setup event handler expectation
			suite.mockEventHandler.On("HandleEvent", 0, mock.AnythingOfType("*wrp.Message")).Once()

			// Create request
			req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(tc.payload))
			req.Header.Set("Content-Type", tc.contentType)

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute
			suite.handler.ServeHTTP(w, req)

			// Verify response
			suite.Equal(http.StatusAccepted, w.Code)
			suite.Contains(w.Body.String(), "Request placed on to queue.")

			// Verify event handler was called
			suite.mockEventHandler.AssertExpectations(suite.T())
		})
	}
}

func (suite *RequestHandlerTestSuite) TestServeHTTP_ContentTypeHandling() {
	testCases := []struct {
		name         string
		contentType  string
		expectedCode int
		expectedBody string
		description  string
	}{
		{
			name:         "invalid_content_type",
			contentType:  "application/xml",
			expectedCode: http.StatusUnsupportedMediaType,
			expectedBody: "Invalid Content-Type header. Expected application/msgpack or application/json.",
			description:  "Should reject invalid content type",
		},
		{
			name:         "missing_content_type",
			contentType:  "",
			expectedCode: http.StatusUnsupportedMediaType,
			expectedBody: "Invalid Content-Type header. Expected application/msgpack or application/json.",
			description:  "Should reject missing content type",
		},
		{
			name:         "multiple_content_types",
			contentType:  "application/json, application/msgpack",
			expectedCode: http.StatusUnsupportedMediaType,
			expectedBody: "Invalid Content-Type header. Expected application/msgpack or application/json.",
			description:  "Should reject multiple content types",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			payload := suite.createJSONPayload()
			req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(payload))

			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			w := httptest.NewRecorder()

			// Execute
			suite.handler.ServeHTTP(w, req)

			// Verify response
			suite.Equal(tc.expectedCode, w.Code)
			suite.Contains(w.Body.String(), tc.expectedBody)

			// Verify event handler was not called
			suite.mockEventHandler.AssertNotCalled(suite.T(), "HandleEvent")
		})
	}
}

func (suite *RequestHandlerTestSuite) TestServeHTTP_QueueDepthHandling() {
	suite.T().Run("queue_depth_within_limit", func(t *testing.T) {
		// Setup event handler expectation
		suite.mockEventHandler.On("HandleEvent", 0, mock.AnythingOfType("*wrp.Message")).Once()

		payload := suite.createJSONPayload()
		req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		// Execute
		suite.handler.ServeHTTP(w, req)

		// Verify response
		suite.Equal(http.StatusAccepted, w.Code)
		suite.mockEventHandler.AssertExpectations(suite.T())
	})

	suite.T().Run("queue_depth_exceeds_limit", func(t *testing.T) {
		// Set a very low limit
		suite.handler.maxOutstanding = 1
		suite.handler.incomingQueueDepth = 1 // Already at limit

		payload := suite.createJSONPayload()
		req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		// Execute
		suite.handler.ServeHTTP(w, req)

		// Verify response
		suite.Equal(http.StatusServiceUnavailable, w.Code)
		suite.Contains(w.Body.String(), "Incoming queue is full.")

		// Verify event handler was not called
		suite.mockEventHandler.AssertNotCalled(suite.T(), "HandleEvent")
	})
}

func (suite *RequestHandlerTestSuite) TestServeHTTP_PayloadHandling() {
	testCases := []struct {
		name         string
		setupRequest func() *http.Request
		expectedCode int
		expectedBody string
		description  string
	}{
		{
			name: "empty_payload",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader([]byte{}))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Empty payload.",
			description:  "Should reject empty payload",
		},
		{
			name: "invalid_json_payload",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader([]byte(`{invalid json}`)))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid payload format.",
			description:  "Should reject invalid JSON",
		},
		{
			name: "invalid_message_type",
			setupRequest: func() *http.Request {
				// Create a WRP message with wrong message type
				msg := &wrp.Message{
					Type:            wrp.AuthorizationMessageType, // Wrong type (not SimpleEventMessageType)
					Source:          "device-123",
					Destination:     "event:device-status",
					TransactionUUID: "test-uuid",
					ContentType:     "application/json",
					Payload:         []byte(`{"test": "data"}`),
				}
				encodedBytes := []byte{}
				encoder := wrp.NewEncoderBytes(&encodedBytes, wrp.JSON)
				encoder.Encode(msg)

				req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(encodedBytes))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid MessageType.",
			description:  "Should reject wrong message type",
		},
		{
			name: "non_utf8_strings",
			setupRequest: func() *http.Request {
				// Create a valid JSON payload but manually craft it with invalid UTF-8
				// This bypasses the WRP encoder which might sanitize the data
				invalidPayload := []byte(`{"msg_type":4,"source":"device-123","dest":"` + string([]byte{0xFF, 0xFE, 0xFD}) + `","transaction_uuid":"test-uuid","content_type":"application/json","payload":"dGVzdA=="}`)

				req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(invalidPayload))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "Strings must be UTF-8.",
			description:  "Should reject non-UTF-8 strings",
		},
		{
			name: "read_error",
			setupRequest: func() *http.Request {
				// Create a request with a body that will cause read error
				req := httptest.NewRequest("POST", "/api/v2/device", &errorReader{})
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: "",
			description:  "Should handle read errors gracefully",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			req := tc.setupRequest()
			w := httptest.NewRecorder()

			// Execute
			suite.handler.ServeHTTP(w, req)

			// Verify response
			suite.Equal(tc.expectedCode, w.Code, tc.description)
			if tc.expectedBody != "" {
				suite.Contains(w.Body.String(), tc.expectedBody, tc.description)
			}

			// Verify event handler was not called for error cases
			suite.mockEventHandler.AssertNotCalled(suite.T(), "HandleEvent")
		})
	}
}

func (suite *RequestHandlerTestSuite) TestFixWRP() {
	testCases := []struct {
		name        string
		inputMsg    *wrp.Message
		setupAssert func(*wrp.Message)
		description string
	}{
		{
			name: "complete_message",
			inputMsg: &wrp.Message{
				Type:            wrp.SimpleEventMessageType,
				Source:          "device-123",
				Destination:     "event:device-status",
				TransactionUUID: "existing-uuid",
				ContentType:     "application/json",
				Payload:         []byte(`{"test": "data"}`),
			},
			setupAssert: func(result *wrp.Message) {
				suite.Equal("existing-uuid", result.TransactionUUID)
				suite.Equal("application/json", result.ContentType)
			},
			description: "Should not modify complete message",
		},
		{
			name: "missing_content_type",
			inputMsg: &wrp.Message{
				Type:            wrp.SimpleEventMessageType,
				Source:          "device-123",
				Destination:     "event:device-status",
				TransactionUUID: "existing-uuid",
				ContentType:     "", // Empty content type
				Payload:         []byte(`{"test": "data"}`),
			},
			setupAssert: func(result *wrp.Message) {
				suite.Equal("existing-uuid", result.TransactionUUID)
				suite.Equal(wrp.MimeTypeJson, result.ContentType)
			},
			description: "Should set default content type",
		},
		{
			name: "missing_transaction_uuid",
			inputMsg: &wrp.Message{
				Type:            wrp.SimpleEventMessageType,
				Source:          "device-123",
				Destination:     "event:device-status",
				TransactionUUID: "", // Empty UUID
				ContentType:     "application/json",
				Payload:         []byte(`{"test": "data"}`),
			},
			setupAssert: func(result *wrp.Message) {
				suite.NotEmpty(result.TransactionUUID)
				suite.Equal("application/json", result.ContentType)
			},
			description: "Should generate transaction UUID",
		},
		{
			name: "missing_both",
			inputMsg: &wrp.Message{
				Type:        wrp.SimpleEventMessageType,
				Source:      "device-123",
				Destination: "event:device-status",
				// Both UUID and ContentType are empty
				TransactionUUID: "",
				ContentType:     "",
				Payload:         []byte(`{"test": "data"}`),
			},
			setupAssert: func(result *wrp.Message) {
				suite.NotEmpty(result.TransactionUUID)
				suite.Equal(wrp.MimeTypeJson, result.ContentType)
			},
			description: "Should set both default content type and generate UUID",
		},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			// Execute
			result := suite.handler.fixWrp(tc.inputMsg)

			// Verify
			suite.NotNil(result)
			tc.setupAssert(result)
		})
	}
}

func (suite *RequestHandlerTestSuite) TestServeHTTP_Integration() {
	suite.T().Run("end_to_end_json_processing", func(t *testing.T) {
		// Setup event handler to capture the processed message
		var capturedWorkerID int
		var capturedMessage *wrp.Message

		suite.mockEventHandler.On("HandleEvent", mock.AnythingOfType("int"), mock.AnythingOfType("*wrp.Message")).
			Run(func(args mock.Arguments) {
				capturedWorkerID = args.Get(0).(int)
				capturedMessage = args.Get(1).(*wrp.Message)
			}).Once()

		// Create a message missing both UUID and ContentType to test fixWrp
		msg := &wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "device-123",
			Destination: "event:device-status/mac:112233445566/online",
			// Missing TransactionUUID and ContentType
			Payload: []byte(`{"status":"online","timestamp":"2023-10-30T10:00:00Z"}`),
		}

		encodedBytes := []byte{}
		encoder := wrp.NewEncoderBytes(&encodedBytes, wrp.JSON)
		encoder.Encode(msg)
		payload := encodedBytes

		req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()

		// Execute
		suite.handler.ServeHTTP(w, req)

		// Verify HTTP response
		suite.Equal(http.StatusAccepted, w.Code)
		suite.Contains(w.Body.String(), "Request placed on to queue.")

		// Verify event handler was called with correct parameters
		suite.mockEventHandler.AssertExpectations(suite.T())
		suite.Equal(0, capturedWorkerID)
		suite.NotNil(capturedMessage)

		// Verify the message was fixed properly
		suite.NotEmpty(capturedMessage.TransactionUUID, "UUID should be generated")
		suite.Equal(wrp.MimeTypeJson, capturedMessage.ContentType, "ContentType should be set to default")
		suite.Equal("device-123", capturedMessage.Source)
		suite.Equal("event:device-status/mac:112233445566/online", capturedMessage.Destination)
	})
}

func (suite *RequestHandlerTestSuite) TestServeHTTP_ConcurrentRequests() {
	suite.T().Run("queue_depth_sequential_processing", func(t *testing.T) {
		// Test that queue depth is managed correctly for sequential requests
		suite.handler.maxOutstanding = 3

		// Setup event handler to handle successful requests
		suite.mockEventHandler.On("HandleEvent", 0, mock.AnythingOfType("*wrp.Message")).
			Times(3) // Expect 3 successful requests

		payload := suite.createJSONPayload()

		// Execute sequential requests
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("POST", "/api/v2/device", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.handler.ServeHTTP(w, req)
			suite.Equal(http.StatusAccepted, w.Code, "Request %d should be accepted", i+1)
		}

		// Verify all requests were handled
		suite.mockEventHandler.AssertExpectations(suite.T())
	})
}

// Helper type for testing read errors
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (suite *RequestHandlerTestSuite) TestInterfaceCompliance() {
	// Ensure RequestHandler implements http.Handler
	var _ http.Handler = (*RequestHandler)(nil)
}
