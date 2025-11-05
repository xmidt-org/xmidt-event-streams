// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"context"
	"fmt"

	"github.com/xmidt-org/xmidt-event-streams/sender"

	"github.com/fogfish/opts"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
)

type DispatcherProvider interface {
	GetDispatcher(f *BaseFilter, d sender.DestType, streams []Stream) (Dispatcher, error)
}

type Dispatcher interface {
	Queue(msg *wrp.Message)
	Shutdown(bool)
}

type DispatcherFactory struct {
	senderProvider sender.SenderProvider
	queueProvider  QueueProvider
	retries        int
	logger         *zap.Logger
	metrics        *FilterMetrics
}

// options
var (
	WithSenderProvider = opts.ForName("senderProvider", func(df *DispatcherFactory, sp sender.SenderProvider) error {
		if sp == nil {
			return fmt.Errorf("senderProvider cannot be nil")
		}
		df.senderProvider = sp
		return nil
	})
	WithQueueProvider = opts.ForName("queueProvider", func(df *DispatcherFactory, qp QueueProvider) error {
		if qp == nil {
			return fmt.Errorf("queueProvider cannot be nil")
		}
		df.queueProvider = qp
		return nil
	})
	WithRetries = opts.ForName("retries", func(df *DispatcherFactory, retries int) error {
		if retries < 0 {
			return fmt.Errorf("retries cannot be negative")
		}
		df.retries = retries
		return nil
	})
	WithDispatchLogger  = opts.ForType[DispatcherFactory, *zap.Logger]()
	WithDispatchMetrics = opts.ForType[DispatcherFactory, *FilterMetrics]()
)

func (df *DispatcherFactory) checkRequired() error {
	return opts.Required(df,
		WithSenderProvider(nil),
		WithQueueProvider(nil),
		WithDispatchLogger(nil),
		WithDispatchMetrics(nil),
	)
}

func NewDispatcherFactory(opt []opts.Option[DispatcherFactory]) (DispatcherProvider, error) {
	df := &DispatcherFactory{}

	if err := opts.Apply(df, opt); err != nil {
		return nil, err
	}

	err := df.checkRequired()
	if err != nil {
		return nil, err
	}

	return df, nil

}

func (df *DispatcherFactory) GetDispatcher(f *BaseFilter, dest sender.DestType, streams []Stream) (Dispatcher, error) {
	// create a sender for each stream
	senders := make([]sender.Sender, 0, len(streams))
	for _, stream := range streams {

		sender, err := df.senderProvider.GetSender(df.retries, dest, stream.GetConfigMap())

		if err != nil {
			return nil, fmt.Errorf("error getting sender for dest type %v: %w", dest, err)
		}

		senders = append(senders, sender)
	}

	df.logger.Debug("Creating dispatcher", zap.Any("senders", senders), zap.String("filterID", f.id), zap.Int("numStreams", len(streams)))

	// create the dispatcher
	dispatcher := &StreamDispatcher{
		senders: senders,
		streams: streams,
		logger:  df.logger,
		metrics: df.metrics,
		id:      f.id,
	}

	// create the dispatcher queue and start it
	queue, err := df.queueProvider.GetQueue(f.id, f.queueSize, f.batchSize, f.maxWorkers, dispatcher)
	if err != nil {
		df.logger.Error("error creating queue", zap.Error(err))
		return nil, err
	}

	dispatcher.queue = queue

	go queue.Start(context.Background())

	return dispatcher, nil
}
