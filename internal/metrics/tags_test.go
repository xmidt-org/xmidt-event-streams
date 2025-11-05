// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetUnknownTagIfEmpty(t *testing.T) {
	assert.Equal(t, "unknown", GetUnknownTagIfEmpty(""))
	assert.Equal(t, "some-tag", GetUnknownTagIfEmpty("some-tag"))
}
