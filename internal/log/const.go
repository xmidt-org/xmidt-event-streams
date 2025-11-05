// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package log

const (
	// elastic search fields
	EventAction     = "event.action"
	ErrorCode       = "error.code"
	ErrorStackTrace = "error.stack_trace"
	ErrorMessage    = "error.message"
	RequestBody     = "http.request.body.content"

	// comcast fields
	Mac      = "device.macAddress"
	Firmware = "device.firmware"
	Model    = "device.model"

	// app specific fields
	SessionId        = "session_id"
	Destination      = "destionation"
	Payload          = "payload"
	Interface        = "interface"
	LastRebootReason = "last_reboot_reason"
	Value            = "value"
	Queue            = "queue"
	Event            = "event"
	Role             = "role"
	EventBody        = "event"
)
