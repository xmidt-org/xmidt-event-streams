// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package integrationTests

import (
	"net/http"
	"testing"
)

func TestPostMatchingEvent(t *testing.T) {
	runIt(t, aTest{
		broken:           false,
		expectedHttpCode: http.StatusOK,
		item: &item{
			dest: "event:device-status/%x/offline",
		},
		outputExpected: true,
	})
}
