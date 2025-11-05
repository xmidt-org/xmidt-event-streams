// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
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
	QueueProvider QueueProvider
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
			var opts []opts.Option[QueueFactory]
			opts = append(opts,
				WithQueueLogger(in.Logger),
				WithQueueTelemetry(in.QueueTelemetry),
			)

			queueProvider, err := NewQueueFactory(opts)

			return QueueOut{
				QueueProvider: queueProvider}, err
		},
	),
)
