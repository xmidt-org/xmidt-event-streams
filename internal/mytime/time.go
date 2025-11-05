// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package mytime

import (
	"errors"
	"strconv"
	"time"
)

var invalidTime = time.Unix(0, 0)

var (
	errInvalidNumericTimeString  = errors.New("invalid numeric time string")
	errInvalidTimeString         = errors.New("invalid time string")
	errInvalidTimeDurationString = errors.New("invalid time duration string")
)

func GetTimeFromXmidtTSString(val string) (time.Time, error) {
	timeParsed, err := time.Parse(time.RFC3339Nano, val)
	return timeParsed, err
}

// get time from number string representing seconds
func GetTimeFromNumberString(val string, duration time.Duration) (time.Time, error) {
	if val != "" {
		t, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return invalidTime, err
		}

		switch duration {
		case time.Second:
			return time.Unix(t, 0).UTC(), nil
		case time.Microsecond:
			return time.UnixMicro(t).UTC(), nil
		}
	}

	return invalidTime, errInvalidNumericTimeString
}

func GetTimeFromDateString(st string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, st)
	if err != nil {
		return invalidTime, errInvalidTimeString
	}
	return t, nil
}

func GetDurationFromString(st string) (time.Duration, error) {
	d, err := time.ParseDuration(st)
	if err != nil {
		return 0, errInvalidTimeDurationString
	}
	return d, nil
}

func TimeToStored(t time.Time) int64 {
	return t.UTC().UnixMicro()
}

func StoredToTime(t int64) time.Time {
	return time.UnixMicro(t).UTC()
}
