// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/xmidt-org/xmidt-event-streams/filter"
	"github.com/xmidt-org/xmidt-event-streams/internal/queue"

	"github.com/fogfish/opts"
	kit "github.com/go-kit/kit/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type QueueTelemetryIn struct {
	fx.In
	QueuedItems  kit.Gauge     `name:"queue_waiting_events"`
	DroppedItems kit.Counter   `name:"queue_dropped_events"`
	BatchSize    kit.Gauge     `name:"queue_batch_size"`
	SubmitErrors kit.Counter   `name:"queue_submit_errors"`
	CallDuration kit.Histogram `name:"queue_submit_duration"`
}

type QueueIn struct {
	fx.In
	Logger         *zap.Logger
	QueueTelemetry *queue.Telemetry
}

type QueueOut struct {
	fx.Out
	QueueProvider filter.QueueProvider
}

var QueueModule = fx.Module("queue",
	fx.Provide(
		func(in QueueTelemetryIn) *queue.Telemetry {
			return &queue.Telemetry{
				QueuedItems:  in.QueuedItems,
				DroppedItems: in.DroppedItems,
				BatchSize:    in.BatchSize,
				SubmitErrors: in.SubmitErrors,
				CallDuration: in.CallDuration,
			}
		}),
	fx.Provide(
		func(in QueueIn) (QueueOut, error) {
			var opts []opts.Option[filter.QueueFactory]
			opts = append(opts,
				filter.WithQueueLogger(in.Logger),
				filter.WithQueueTelemetry(in.QueueTelemetry),
			)

			queueProvider, err := filter.NewQueueFactory(opts)

			return QueueOut{
				QueueProvider: queueProvider}, err
		},
	),
)
