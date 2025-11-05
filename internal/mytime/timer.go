// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package mytime

import (
	"time"

	kit "github.com/go-kit/kit/metrics"
)

func Timer(name string, hist kit.Histogram) func() {
	start := time.Now()
	return func() {
		hist.With("name", name).Observe(float64(time.Since(start).Milliseconds()))
	}
}
