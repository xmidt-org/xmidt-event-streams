// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package filter

import (
	kit "github.com/go-kit/kit/metrics"
)

const (
	// metric labels
	codeLabel   = "code"
	urlLabel    = "url"
	eventLabel  = "event"
	reasonLabel = "reason"
)

type FilterManagerMetrics struct {
	EventType kit.Counter
}

type FilterMetrics struct {
	DroppedMessage kit.Counter `name:"dropped_message_counter"`
}
