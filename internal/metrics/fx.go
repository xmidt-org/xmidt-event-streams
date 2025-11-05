// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone/touchkit"
	"go.uber.org/fx"
)

type metricType int

const (
	COUNTER           metricType = 1
	GAUGE             metricType = 2
	HISTOGRAM         metricType = 3
	COUNTER_NO_LABELS metricType = 4
	GAUGE_NO_LABELS   metricType = 5
)

type metricDefinition struct {
	Type    metricType
	Name    string // the metric name (prometheus.CounterOpts.Name, etc)
	Help    string // the metric help (prometheus.CounterOpts.Help, etc)
	Labels  string // a comma separated list of labels that are whitespace trimmed
	Buckets string // a comma separated list of labels that are whitespace trimmed
}

const (
	CodeLabel   = "code"
	UrlLabel    = "url"
	EventLabel  = "event"
	ReasonLabel = "reason"
)

var fxMetrics = []metricDefinition{
	{
		Type:   GAUGE,
		Name:   "queue_waiting_events",
		Help:   "The number of events waiting to run in the queue.",
		Labels: "name",
	},
	{
		Type:   GAUGE,
		Name:   "queue_batch_size",
		Help:   "The number of items in a batch submitted for processing.",
		Labels: "name",
	},
	{
		Type:   COUNTER,
		Name:   "queue_dropped_events",
		Help:   "The number of events dropped on the floor.",
		Labels: "name",
	},
	{
		Type:   COUNTER,
		Name:   "queue_submit_errors",
		Help:   "error when queue submits a batch",
		Labels: "name",
	}, {
		Type:    HISTOGRAM,
		Name:    "queue_submit_duration",
		Help:    "duration of call",
		Labels:  "name",
		Buckets: "10, 100, 1000, 5000, 10000, 100000, 500000, 1000000, 2000000",
	}, {
		Type:   COUNTER,
		Name:   "event_type_count",
		Help:   "Incoming count of events by event type",
		Labels: EventLabel,
	}, {
		Type:   COUNTER,
		Name:   "dropped_message_count",
		Help:   "messages dropped by id",
		Labels: fmt.Sprintf("%s,%s", UrlLabel, ReasonLabel),
	},
}

func Provide() fx.Option {
	opts := make([]fx.Option, 0, 10)

	for _, m := range fxMetrics {
		labels := strings.Split(m.Labels, ",")
		for i := range labels {
			labels[i] = strings.TrimSpace(labels[i])
		}

		var opt fx.Option

		switch m.Type {
		case COUNTER:
			opt = touchkit.Counter(
				prometheus.CounterOpts{
					Name: m.Name,
					Help: m.Help,
				},
				labels...)
		case COUNTER_NO_LABELS:
			opt = touchkit.Counter(
				prometheus.CounterOpts{
					Name: m.Name,
					Help: m.Help,
				},
			)
		case GAUGE:
			opt = touchkit.Gauge(
				prometheus.GaugeOpts{
					Name: m.Name,
					Help: m.Help,
				},
				labels...)
		case GAUGE_NO_LABELS:
			opt = touchkit.Gauge(
				prometheus.GaugeOpts{
					Name: m.Name,
					Help: m.Help,
				},
			)
		case HISTOGRAM:
			buckets := strings.Split(m.Buckets, ",")
			bucketLimits := make([]float64, len(buckets))
			for i := range buckets {
				bucketLimit, err := strconv.ParseFloat(strings.TrimSpace(buckets[i]), 64)
				if err != nil {
					panic(fmt.Sprintf("bucket has non-numeric value '%s'", buckets[i]))
				}
				bucketLimits[i] = bucketLimit
			}
			opt = touchkit.Histogram(
				prometheus.HistogramOpts{
					Name:    m.Name,
					Help:    m.Help,
					Buckets: bucketLimits,
				},
				labels...)
		default:
			panic(fmt.Sprintf("unknown metric type %d for '%s'", m.Type, m.Name))
		}

		if opt == nil {
			panic(fmt.Sprintf("failed to create metric '%s'", m.Name))
		}

		opts = append(opts, opt)
	}

	return fx.Options(opts...)
}
