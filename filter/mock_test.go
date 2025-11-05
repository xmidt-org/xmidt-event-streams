// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"context"

	"github.com/xmidt-org/xmidt-event-streams/internal/queue"
	"github.com/xmidt-org/xmidt-event-streams/sender"

	kit "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/wrp-go/v3"
)

// NewMockCounter creates a new mock counter for testing
func NewMockCounter() *MockCounter {
	return &MockCounter{}
}

// MockCounter implements the Counter interface for testing
type MockCounter struct {
	mock.Mock
}

func (m *MockCounter) With(label ...string) kit.Counter {
	args := m.Called(label)
	return args.Get(0).(kit.Counter)
}

func (m *MockCounter) Add(delta float64) {
	m.Called(delta)
}

// NewMockGauge creates a new mock gauge for testing
func NewMockGauge() *MockGauge {
	return &MockGauge{}
}

// MockGauge implements the Gauge interface for testing
type MockGauge struct {
	mock.Mock
}

func (m *MockGauge) With(label ...string) kit.Gauge {
	args := m.Called(label)
	return args.Get(0).(kit.Gauge)
}

func (m *MockGauge) Set(value float64) {
	m.Called(value)
}

func (m *MockGauge) Add(delta float64) {
	m.Called(delta)
}

// NewMockHistogram creates a new mock histogram for testing
func NewMockHistogram() *MockHistogram {
	return &MockHistogram{}
}

// MockHistogram implements the Histogram interface for testing
type MockHistogram struct {
	mock.Mock
}

func (m *MockHistogram) With(label ...string) kit.Histogram {
	args := m.Called(label)
	return args.Get(0).(kit.Histogram)
}

func (m *MockHistogram) Observe(value float64) {
	m.Called(value)
}

// MockQueue implements the queue.Queue interface for testing
type MockQueue struct {
	mock.Mock
}

func NewMockQueue() *MockQueue { return &MockQueue{} }

func (m *MockQueue) ProcessItems(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Get(0).(bool)
}

func (m *MockQueue) Start(ctx context.Context) {
	m.Called(ctx)
}

func (m *MockQueue) AddItem(msg *wrp.Message) {
	m.Called(msg)
}

func (m *MockQueue) SetBatchSize(size int) {
	m.Called(size)
}

func (m *MockQueue) Close() {
	m.Called()
}

func (m *MockQueue) Empty() {
	m.Called()
}

func (m *MockQueue) Wait() {
	m.Called()
}

// MockQueueProvider implements the QueueProvider interface for testing
type MockQueueProvider struct {
	mock.Mock
}

func NewMockQueueProvider() *MockQueueProvider {
	return &MockQueueProvider{}
}

func (m *MockQueueProvider) GetQueue(name string, queueSize int, batchSize int, maxWorkers int, submitter queue.Submitter[*wrp.Message]) (queue.Queue[*wrp.Message], error) {
	args := m.Called(name, queueSize, batchSize, maxWorkers, submitter)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(queue.Queue[*wrp.Message]), args.Error(1)
}

// MockDispatcherProvider implements the DispatcherProvider interface for testing
type MockDispatcherProvider struct {
	mock.Mock
}

func (m *MockDispatcherProvider) GetDispatcher(f *BaseFilter, t sender.DestType, streams []Stream) (Dispatcher, error) {
	args := m.Called(f, t, streams)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(Dispatcher), args.Error(1)
}

// MockFilter implements the Filter interface for testing
type MockFilter struct {
	mock.Mock
}

func (m *MockFilter) Configure(cfg FilterConfig) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *MockFilter) Shutdown(gentle bool) {
	m.Called(gentle)
}

func (m *MockFilter) Queue(msg *wrp.Message) {
	m.Called(msg)
}

// MockSenderProvider implements the sender.SenderProvider interface for testing
type MockSenderProvider struct {
	mock.Mock
}

func (m *MockSenderProvider) GetSender(retries int, d sender.DestType, config map[string]string) (sender.Sender, error) {
	args := m.Called(retries, d, config)
	result := args.Get(0)
	if result == nil {
		return nil, args.Error(1)
	}
	return result.(sender.Sender), args.Error(1)
}

// MockSender implements the sender.Sender interface for testing
type MockSender struct {
	mock.Mock
}

func (m *MockSender) OnEvent(msgs []*wrp.Message, stream string) (int, error) {
	args := m.Called(msgs, stream)
	return args.Int(0), args.Error(1)
}

// MockDispatcher implements the Dispatcher interface for testing
type MockDispatcher struct {
	mock.Mock
}

func (m *MockDispatcher) Queue(msg *wrp.Message) {
	m.Called(msg)
}

func (m *MockDispatcher) Shutdown(gentle bool) {
	m.Called(gentle)
}
