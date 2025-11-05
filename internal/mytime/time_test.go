// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package mytime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTimeFromDateString(t *testing.T) {
	result, err := GetTimeFromDateString("")
	assert.Equal(t, errInvalidTimeString, err)
	assert.Equal(t, int64(0), result.Unix())

	result, err = GetTimeFromDateString("2016-11-01T20:44:39Z")
	assert.Nil(t, err)
	assert.Equal(t, int64(1478033079), result.Unix())
}

func TestGetTimeFromNumberString(t *testing.T) {
	result, err := GetTimeFromNumberString("", time.Second)
	assert.NotNil(t, err)
	assert.Equal(t, time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC), result.UTC())

	result, err = GetTimeFromNumberString("1478033079", time.Second)
	assert.Nil(t, err)
	assert.Equal(t, time.Date(2016, time.November, 1, 20, 44, 39, 0, time.UTC), result.UTC())

	result, err = GetTimeFromNumberString("1478033079000011", time.Microsecond)
	assert.Nil(t, err)
	assert.Equal(t, time.Date(2016, time.November, 1, 20, 44, 39, 11000, time.UTC), result.UTC())
}
