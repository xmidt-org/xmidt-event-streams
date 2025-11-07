// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package filter

import (
	"strings"

	"errors"
	"fmt"
	"regexp"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"github.com/xmidt-org/xmidt-event-streams/internal/sender"
)

// BaseFilterFactory is a configurable factory for BaseFilter objects.
type BaseFilterFactory struct {
	// filter regex and endpoint configuration
	FilterConfig FilterConfig

	// The number of delivery workers to create and use.
	NumWorkers int

	// The queue depth to buffer events before we start dropping new messages
	defaultQueueSize  int
	defaultMaxWorkers int
	defaultBatchSize  int

	// Number of delivery retries before giving up
	DeliveryRetries int

	// Time in between delivery retries
	//DeliveryInterval time.Duration

	// filter metrics
	Metrics *FilterMetrics

	// The logger to use.
	Logger *zap.Logger

	DispatcherProvider DispatcherProvider
}

// New creates a new filter from the factory, or returns an error.
func (ff BaseFilterFactory) New(cfg FilterConfig) (Filter, error) {
	decoratedLogger := ff.Logger.With(zap.String("id", cfg.Stream.StreamName))

	// create filter
	f := &BaseFilter{
		id: strings.ToValidUTF8(cfg.Stream.StreamName, ""),
		//partnerIds: cfg.PartnerIds,

		queueSize:       cfg.QueueSize,
		batchSize:       cfg.BatchSize,
		maxWorkers:      cfg.MaxWorkers,
		streamVersion:   cfg.StreamVersion,
		logger:          decoratedLogger,
		deliveryRetries: ff.DeliveryRetries,
		metrics:         ff.Metrics,
	}

	// configure queue constraints
	err := ff.configureQueue(f, cfg)
	if err != nil {
		return nil, fmt.Errorf("error configuring queue for filter %s: %w", f.id, err)
	}

	// configure dispatcher
	err = ff.configureDispatcher(f, cfg)
	if err != nil {
		return nil, fmt.Errorf("error configuring dispatcher for filter %s: %w", f.id, err)
	}

	// configure matchers
	err = ff.configureMatchers(f, cfg)
	if err != nil {
		return nil, fmt.Errorf("error configuring matchers for filter %s: %w", f.id, err)
	}

	f.wg.Add(1)

	return f, nil
}

func (ff *BaseFilterFactory) configureQueue(f *BaseFilter, cfg FilterConfig) error {
	if cfg.BatchSize <= 0 {
		f.batchSize = ff.defaultBatchSize
	} else {
		f.batchSize = cfg.BatchSize
	}

	if cfg.QueueSize <= 0 {
		f.queueSize = ff.defaultQueueSize
	} else {
		f.queueSize = cfg.QueueSize
	}

	if cfg.MaxWorkers <= 0 {
		f.maxWorkers = ff.defaultMaxWorkers
	} else {
		f.maxWorkers = cfg.MaxWorkers
	}
	f.workers = semaphore.NewWeighted(int64(f.maxWorkers))

	return nil
}

func (ff *BaseFilterFactory) configureDispatcher(f *BaseFilter, cfg FilterConfig) error {
	streams := []Stream{}
	streams = append(streams, cfg.Stream)
	streams = append(streams, cfg.AltStreams...)

	destEnum, err := sender.ParseDestType(cfg.DestType)
	if err != nil {
		return fmt.Errorf("error parsing destination type %s", cfg.DestType)
	}

	f.dispatcher, err = ff.DispatcherProvider.GetDispatcher(f, destEnum, streams)
	if err != nil {
		f.logger.Error("error creating dispatcher for filter", zap.String("ID", f.id), zap.Error(err))
		return err
	}

	return nil
}

func (ff *BaseFilterFactory) configureMatchers(f *BaseFilter, cfg FilterConfig) error {
	eventMatchers := make([]*regexp.Regexp, 0, len(cfg.Events))
	for _, event := range cfg.Events {
		var re *regexp.Regexp
		var err error
		if re, err = regexp.Compile(event); err != nil {
			return fmt.Errorf("invalid event matcher item: '%s'", event)
		}

		eventMatchers = append(eventMatchers, re)
	}
	if len(eventMatchers) < 1 {
		return errors.New("event matchers must not be empty")
	}
	f.eventMatchers = eventMatchers

	metaMatchers := make([]*regexp.Regexp, 0, len(cfg.Metadata.DeviceIds))
	for _, item := range cfg.Metadata.DeviceIds {
		if item == ".*" {
			// Match everything - skip the filtering
			metaMatchers = []*regexp.Regexp{}
			break
		}

		var re *regexp.Regexp
		var err error
		if re, err = regexp.Compile(item); err != nil {
			err = fmt.Errorf("invalid matcher item: '%s'", item)
			return err
		}
		metaMatchers = append(metaMatchers, re)
	}
	// if matcher list is empty set it nil for Queue() logic
	if len(metaMatchers) == 0 {
		metaMatchers = nil
	}

	f.metadataMatchers = metaMatchers

	return nil
}
