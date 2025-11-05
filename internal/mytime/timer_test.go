// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package mytime

import (
	"testing"

	kit "github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/mock"
)

type mockHistogram struct {
	mock.Mock
}

func newMockHistogram() *mockHistogram { return &mockHistogram{} }

func (m *mockHistogram) With(label ...string) kit.Histogram {
	m.Called(label)
	return m
}

func (m *mockHistogram) Observe(amount float64) {
	m.Called(amount)
}

func TestCallDurationTimer(t *testing.T) {
	hist := newMockHistogram()
	hist.On("Observe", mock.Anything).On("With", mock.Anything)

	someMethod(hist)

	hist.AssertCalled(t, "Observe", mock.Anything)
}

func someMethod(hist kit.Histogram) {
	defer Timer("someMethod", hist)()
}
