// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package filter

type FilterConfig struct {
	Stream        Stream
	AltStreams    []Stream
	Events        []string
	Metadata      Metadata
	DestType      string
	QueueSize     int
	BatchSize     int
	MaxWorkers    int
	StreamVersion string
}

type Metadata struct {
	// DeviceIds is the list of regular expressions to match device id against.
	DeviceIds []string
}

type Stream struct {
	StreamName string
	Config     map[string]string
}
