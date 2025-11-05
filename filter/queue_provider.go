// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"eventstream/internal/queue"

	"github.com/fogfish/opts"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
)

type QueueProvider interface {
	GetQueue(name string, queueSize int, batchSize int, maxWorkers int, submitter queue.Submitter[*wrp.Message]) (queue.Queue[*wrp.Message], error)
}

type QueueFactory struct {
	t      *queue.Telemetry
	logger *zap.Logger
}

// options
var (
	WithQueueTelemetry = opts.ForType[QueueFactory, *queue.Telemetry]()
	WithQueueLogger    = opts.ForType[QueueFactory, *zap.Logger]()
)

func (qf *QueueFactory) checkRequired() error {
	return opts.Required(qf,
		WithQueueTelemetry(nil),
		WithQueueLogger(nil),
	)
}

func NewQueueFactory(opt []opts.Option[QueueFactory]) (QueueProvider, error) {
	qf := &QueueFactory{}

	if err := opts.Apply(qf, opt); err != nil {
		return nil, err
	}
	if err := qf.checkRequired(); err != nil {
		return nil, err
	}
	return qf, nil
}

func (qf *QueueFactory) GetQueue(name string, queueSize int, batchSize int, maxWorkers int, submitter queue.Submitter[*wrp.Message]) (queue.Queue[*wrp.Message], error) {
	cfg := queue.QueueConfig{
		ChannelSize:    queueSize,
		BatchSize:      batchSize,
		WorkerPoolSize: maxWorkers,
		Name:           name,
	}
	return queue.NewQueue(
		cfg,
		qf.t,
		qf.logger,
		submitter,
	)
}
