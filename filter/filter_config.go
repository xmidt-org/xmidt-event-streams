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

type ConfigItem struct {
	Key   string
	Value string
}

type Metadata struct {
	// DeviceIds is the list of regular expressions to match device id against.
	DeviceIds []string
}

type Stream struct {
	StreamName  string
	ConfigItems []ConfigItem
}

func (s Stream) GetConfigItem(key string) (string, bool) {
	for _, item := range s.ConfigItems {
		if item.Key == key {
			return item.Value, true
		}
	}
	return "", false
}

func (s Stream) GetConfigMap() map[string]string {
	configMap := make(map[string]string)
	for _, item := range s.ConfigItems {
		configMap[item.Key] = item.Value
	}
	return configMap
}
