// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0


package filter

import (
	
	"github.com/xmidt-org/xmidt-event-streams/sender"

	"github.com/fogfish/opts"
	kit "github.com/go-kit/kit/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FilterTelemetryIn struct {
	fx.In
	DroppedMessage kit.Counter `name:"xmidt_event_streams_dropped_message_count"`
}

type FilterManagerTelemetryIn struct {
	fx.In
	EventType kit.Counter `name:"xmidt_event_streams_event_type_count"`
}

type FilterManagerConfig struct {
	DeliveryRetries   int
	Filters           []FilterConfig
	DefaultQueueSize  int
	DefaultBatchSize  int
	DefaultMaxWorkers int
}

type FilterManagerIn struct {
	fx.In
	Logger               *zap.Logger
	FilterManager        FilterManagerConfig
	FilterMetrics        *FilterMetrics
	FilterManagerMetrics *FilterManagerMetrics
	QueueProvider        QueueProvider
	SenderProvider       sender.SenderProvider
}

type FilterManagerOut struct {
	fx.Out
	FilterManager FilterManager
}

var FilterModule = fx.Module("filter",
	fx.Provide(
		func(in FilterTelemetryIn) *FilterMetrics {
			return &FilterMetrics{
				DroppedMessage: in.DroppedMessage,
			}
		}),
	fx.Provide(
		func(in FilterManagerTelemetryIn) *FilterManagerMetrics {
			return &FilterManagerMetrics{
				EventType: in.EventType,
			}
		}),
	fx.Provide(
		func(in FilterManagerIn) (FilterManagerOut, error) {
			var dispatcherOpts []opts.Option[DispatcherFactory]
			dispatcherOpts = append(dispatcherOpts,
				WithQueueProvider(in.QueueProvider),
				WithSenderProvider(in.SenderProvider),
				WithDispatchLogger(in.Logger),
				WithDispatchMetrics(in.FilterMetrics),
			)

			dispatcherProvider, err := NewDispatcherFactory(dispatcherOpts)
			if err != nil {
				return FilterManagerOut{}, err
			}

			var opts []opts.Option[BaseFilterManager]
			opts = append(opts,
				WithDeliveryRetries(in.FilterManager.DeliveryRetries),
				WithLogger(in.Logger),
				WithFilterMetrics(in.FilterMetrics),
				WithFilterManagerMetrics(in.FilterManagerMetrics),
				WithFilters(in.FilterManager.Filters),
				WithDefaultBatchSize(in.FilterManager.DefaultBatchSize),
				WithDefaultQueueSize(in.FilterManager.DefaultQueueSize),
				WithDefaultWorkers(in.FilterManager.DefaultMaxWorkers),
				WithDeliveryRetries(in.FilterManager.DeliveryRetries),
				WithDispatchProvider(dispatcherProvider),
			)

			manager, err := New(opts)
			if err != nil {
				return FilterManagerOut{}, err
			}

			return FilterManagerOut{
				FilterManager: manager}, err
		},
	),
)
