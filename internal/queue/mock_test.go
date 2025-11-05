// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

//coverage:ignore file
package queue

import (
	kit "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/mock"
)

// mocks shared by multiple packages

// metrics

type MockGauge struct {
	mock.Mock
}

func NewMockGauge() *MockGauge { return &MockGauge{} }

func (m *MockGauge) With(label ...string) kit.Gauge {
	m.Called(label)
	return m
}

func (m *MockGauge) Add(amount float64) {
	m.Called(amount)
}

func (m *MockGauge) Set(amount float64) {
	m.Called(amount)
}

type MockHistogram struct {
	mock.Mock
}

func NewMockHistogram() *MockHistogram { return &MockHistogram{} }

func (m *MockHistogram) With(label ...string) kit.Histogram {
	m.Called(label)
	return m
}

func (m *MockHistogram) Observe(amount float64) {
	m.Called(amount)
}

type MockCounter struct {
	mock.Mock
}

func NewMockCounter() *MockCounter { return &MockCounter{} }

func (m *MockCounter) With(label ...string) kit.Counter {
	m.Called(label)
	return m
}

func (m *MockCounter) Add(delta float64) {
	m.Called(delta)
}
