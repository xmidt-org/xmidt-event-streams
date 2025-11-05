// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package filter

import (
	"fmt"
	"sync"

	"github.com/fogfish/opts"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
)

type FilterManager interface {
	Queue(*wrp.Message)
	Shutdown(bool)
}

type BaseFilterManager struct {
	dispatchProvider  DispatcherProvider
	deliveryRetries   int
	logger            *zap.Logger
	mutex             sync.RWMutex
	filters           map[string]Filter
	metrics           *FilterManagerMetrics
	filterMetrics     *FilterMetrics
	wg                sync.WaitGroup
	shutdown          chan struct{}
	customPIDs        []string
	disablePartnerIds bool
	filterConfig      []FilterConfig
	defaultQueueSize  int
	defaultBatchSize  int
	defaultMaxWorkers int
}

var (
	WithDispatchProvider     = opts.ForType[BaseFilterManager, DispatcherProvider]()
	WithFilters              = opts.ForType[BaseFilterManager, []FilterConfig]()
	WithDeliveryRetries      = opts.ForName[BaseFilterManager, int]("deliveryRetries")
	WithLogger               = opts.ForType[BaseFilterManager, *zap.Logger]()
	WithFilterMetrics        = opts.ForType[BaseFilterManager, *FilterMetrics]()
	WithFilterManagerMetrics = opts.ForType[BaseFilterManager, *FilterManagerMetrics]()
	WithDefaultQueueSize     = opts.ForName[BaseFilterManager, int]("defaultQueueSize")
	WithDefaultBatchSize     = opts.ForName[BaseFilterManager, int]("defaultBatchSize")
	WithDefaultWorkers       = opts.ForName[BaseFilterManager, int]("defaultMaxWorkers")
)

func (fm *BaseFilterManager) checkRequired() error {
	return opts.Required(fm,
		WithLogger(nil),
		WithFilterMetrics(nil),
		WithFilterManagerMetrics(nil),
		WithDispatchProvider(nil),
		WithDefaultQueueSize(0),
		WithDefaultBatchSize(0),
		WithDefaultWorkers(0),
	)
}

func New(opt []opts.Option[BaseFilterManager]) (FilterManager, error) {
	bfw := &BaseFilterManager{
		filters:  make(map[string]Filter),
		shutdown: make(chan struct{}),
	}

	if err := opts.Apply(bfw, opt); err != nil {
		return nil, err
	}

	err := bfw.checkRequired()
	if err != nil {
		return bfw, err
	}

	bfw.wg.Add(1)

	err = bfw.loadFilters()

	return bfw, err
}

func (fm *BaseFilterManager) loadFilters() error {

	ff := BaseFilterFactory{
		DeliveryRetries:    fm.deliveryRetries,
		Metrics:            fm.filterMetrics,
		Logger:             fm.logger,
		DispatcherProvider: fm.dispatchProvider,
		defaultQueueSize:   fm.defaultQueueSize,
		defaultMaxWorkers:  fm.defaultMaxWorkers,
		defaultBatchSize:   fm.defaultBatchSize,
	}

	ids := make([]struct {
		config FilterConfig
		ID     string
	}, len(fm.filterConfig))

	for i, cfg := range fm.filterConfig {
		fm.logger.Debug("Creating filter with config:", zap.Any("config", cfg))
		ids[i].config = cfg
		ids[i].ID = cfg.Stream.StreamName
		f, err := ff.New(cfg)
		if err != nil {
			fm.logger.Error("error creating filter", zap.String("id", ids[i].ID), zap.Error(err))
			return err
		}
		fm.filters[ids[i].ID] = f
	}

	return nil
}

// Queue is used to send all the possible streams a message
func (fm *BaseFilterManager) Queue(msg *wrp.Message) {
	if fm.logger == nil {
		fmt.Println("logger is nil")
	}

	if msg == nil {
		fm.logger.Warn("nil message received, dropping")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			fm.logger.Error("panic recovered", zap.Any("panic", r))
		}
	}()

	fm.metrics.EventType.With(eventLabel, msg.FindEventStringSubMatch()).Add(1)

	fm.logger.Debug("filter manager queuing message", zap.Any("msg", msg))

	for _, v := range fm.filters {
		v.Queue(msg)
	}
}

// Shutdown closes down the delivery mechanisms and cleans up the underlying
// OutboundSenders either gently (waiting for delivery queues to empty) or not
// (dropping enqueued messages)
func (fm *BaseFilterManager) Shutdown(gentle bool) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	for k, v := range fm.filters {
		v.Shutdown(gentle)
		delete(fm.filters, k)
	}
	close(fm.shutdown)
}
