// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"eventstream/filter"

	"eventstream/sender"

	"github.com/fogfish/opts"
	kit "github.com/go-kit/kit/metrics"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FilterTelemetryIn struct {
	fx.In
	DroppedMessage kit.Counter `name:"dropped_message_count"`
}

type FilterManagerTelemetryIn struct {
	fx.In
	EventType kit.Counter `name:"event_type_count"`
}

type FilterManagerIn struct {
	fx.In
	Logger               *zap.Logger
	FilterManager        FilterManager
	FilterMetrics        *filter.FilterMetrics
	FilterManagerMetrics *filter.FilterManagerMetrics
	QueueProvider        filter.QueueProvider
	SenderProvider       sender.SenderProvider
}

type FilterManagerOut struct {
	fx.Out
	FilterManager filter.FilterManager
}

var FilterModule = fx.Module("filter",
	fx.Provide(
		func(in FilterTelemetryIn) *filter.FilterMetrics {
			return &filter.FilterMetrics{
				DroppedMessage: in.DroppedMessage,
			}
		}),
	fx.Provide(
		func(in FilterManagerTelemetryIn) *filter.FilterManagerMetrics {
			return &filter.FilterManagerMetrics{
				EventType: in.EventType,
			}
		}),
	fx.Provide(
		func(in FilterManagerIn) (FilterManagerOut, error) {
			var dispatcherOpts []opts.Option[filter.DispatcherFactory]
			dispatcherOpts = append(dispatcherOpts,
				filter.WithQueueProvider(in.QueueProvider),
				filter.WithSenderProvider(in.SenderProvider),
				filter.WithDispatchLogger(in.Logger),
				filter.WithDispatchMetrics(in.FilterMetrics),
			)

			dispatcherProvider, err := filter.NewDispatcherFactory(dispatcherOpts)
			if err != nil {
				return FilterManagerOut{}, err
			}

			var opts []opts.Option[filter.BaseFilterManager]
			opts = append(opts,
				filter.WithDeliveryRetries(in.FilterManager.DeliveryRetries),
				filter.WithLogger(in.Logger),
				filter.WithFilterMetrics(in.FilterMetrics),
				filter.WithFilterManagerMetrics(in.FilterManagerMetrics),
				filter.WithFilters(in.FilterManager.Filters),
				filter.WithDefaultBatchSize(in.FilterManager.DefaultBatchSize),
				filter.WithDefaultQueueSize(in.FilterManager.DefaultQueueSize),
				filter.WithDefaultWorkers(in.FilterManager.DefaultMaxWorkers),
				filter.WithDeliveryRetries(in.FilterManager.DeliveryRetries),
				filter.WithDispatchProvider(dispatcherProvider),
			)

			manager, err := filter.New(opts)
			if err != nil {
				return FilterManagerOut{}, err
			}

			return FilterManagerOut{
				FilterManager: manager}, err
		},
	),
)
