// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package event

import (
	"github.com/stretchr/testify/mock"

	"github.com/xmidt-org/wrp-go/v3"
)

// mockHandler only needs to mock the `HandleEvent` method
type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) HandleEvent(workerID int, msg *wrp.Message) {
	m.Called(workerID, msg)
}
