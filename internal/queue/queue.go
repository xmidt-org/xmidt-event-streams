// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package queue

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/xmidt-org/xmidt-event-streams/internal/log"
	"github.com/xmidt-org/xmidt-event-streams/internal/mytime"

	kit "github.com/go-kit/kit/metrics"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

const (
	DefaultBatchSize             = 10
	DefaultWorkerPoolSize        = 10
	DefaultChannelSize           = 100
	DefaultBatchTimeLimitSeconds = 10
)

type Telemetry struct {
	QueuedItems  kit.Gauge
	DroppedItems kit.Counter
	BatchSize    kit.Gauge
	SubmitErrors kit.Counter
	CallDuration kit.Histogram
}

type QueueConfig struct {
	Name                  string
	ChannelSize           int
	WorkerPoolSize        int
	BatchSize             int
	BatchTimeLimitSeconds int
	SubmitOnEmptyQueue    bool
}

type Submitter[T any] interface {
	Submit(items []T) error
}

type Queue[T any] interface {
	Start(ctx context.Context)
	AddItem(item T)
	SetBatchSize(size int)
	Close()
	Empty()
	Wait()
}

type MyQueue[T any] struct {
	items          chan T
	sem            *semaphore.Weighted
	logger         *zap.Logger
	t              *Telemetry
	batchSubmitter Submitter[T]
	wg             sync.WaitGroup
	cfg            *QueueConfig
	started        bool
	mutex          sync.RWMutex
}

// we can't use options here beacause of generics
func NewQueue[T any](cfg QueueConfig, t *Telemetry, logger *zap.Logger, batchSubmitter Submitter[T]) (Queue[T], error) {
	if logger == nil {
		return nil, errors.New("logger is required")
	}

	if t == nil {
		return nil, errors.New("telemetry is required")
	}

	if batchSubmitter == nil || (reflect.ValueOf(batchSubmitter).Kind() == reflect.Ptr && reflect.ValueOf(batchSubmitter).IsNil()) {
		return nil, errors.New("batchSubmitter is required")
	}

	if cfg.Name == "" {
		return nil, errors.New("queue name is required")
	}

	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if cfg.ChannelSize <= 0 {
		cfg.ChannelSize = DefaultChannelSize
	}
	if cfg.WorkerPoolSize <= 0 {
		cfg.WorkerPoolSize = DefaultWorkerPoolSize
	}
	if cfg.BatchTimeLimitSeconds <= 0 {
		cfg.BatchTimeLimitSeconds = DefaultBatchTimeLimitSeconds
	}
	q := &MyQueue[T]{
		items:          make(chan T, cfg.ChannelSize),
		sem:            semaphore.NewWeighted(int64(cfg.WorkerPoolSize)),
		logger:         logger,
		t:              t,
		cfg:            &cfg,
		batchSubmitter: batchSubmitter,
	}

	logger.Info("queue config", zap.Int("batchSize", cfg.BatchSize), zap.Int("channelSize", cfg.ChannelSize), zap.Int("workerPoolSize", cfg.WorkerPoolSize), zap.Bool("submitOnEmptyQueue", cfg.SubmitOnEmptyQueue), zap.Int("batchTimeLimitSeconds", cfg.BatchTimeLimitSeconds))

	return q, nil
}

func (q *MyQueue[T]) Close() {
	close(q.items)
}

func (q *MyQueue[T]) Empty() {
	numDropped := len(q.items)
	q.items = make(chan T, q.cfg.ChannelSize)
	q.t.DroppedItems.With("name", q.cfg.Name).Add(float64(numDropped))
	q.t.QueuedItems.With("name", q.cfg.Name).Set(0.0)
}

func (q *MyQueue[T]) Wait() {
	q.wg.Wait()
}

func (q *MyQueue[T]) SetBatchSize(size int) {
	q.cfg.BatchSize = size
}

func (q *MyQueue[T]) AddItem(item T) {
	if len(q.items) < cap(q.items) {
		q.items <- item
		return
	}

	q.t.DroppedItems.With("name", q.cfg.Name).Add(1)
}

func (q *MyQueue[T]) Start(ctx context.Context) {
	q.mutex.Lock()
	if q.started {
		q.mutex.Unlock()
		return
	}
	q.started = true
	q.mutex.Unlock()

	q.wg.Add(1)
	defer q.wg.Done()

	items := []T{}
	timeout := time.NewTicker(time.Duration(q.cfg.BatchTimeLimitSeconds) * time.Second)

	for {
		select {
		case <-ctx.Done():
			q.logger.Info("Context canceled, ending queue work", zap.String(log.Queue, q.cfg.Name), zap.Error(ctx.Err()))
			return
		case item := <-q.items:
			q.mutex.Lock()
			items = append(items, item)
			q.mutex.Unlock()
			q.t.QueuedItems.With("name", q.cfg.Name).Set(float64(len(q.items)))
			q.logger.Debug("queue size", zap.Int("size", len(q.items)))
			q.processItems(ctx, &items)
		case <-timeout.C:
			q.logger.Debug("time limit reached, processing items", zap.Int("currentBatchSize", len(items)))
			q.submitItems(ctx, &items)
			timeout = time.NewTicker(time.Duration(q.cfg.BatchTimeLimitSeconds) * time.Second)
		}
	}
}

func (q *MyQueue[T]) processItems(ctx context.Context, items *[]T) {
	q.logger.Debug("processing items", zap.Int("currentBatchSize", len(*items)))
	if len(*items) >= q.cfg.BatchSize || q.submitIfQueueIsEmpty() {
		q.submitItems(ctx, items)
	}
}

func (q *MyQueue[T]) submitItems(ctx context.Context, items *[]T) {
	defer mytime.Timer(q.cfg.Name, q.t.CallDuration)()

	if len(*items) == 0 {
		return
	}

	if err := q.sem.Acquire(ctx, int64(1)); err == nil {
		itemsToSubmit := *items
		go func() {
			defer q.sem.Release(int64(1))
			q.t.BatchSize.With("name", q.cfg.Name).Set(float64(len(itemsToSubmit)))
			q.logger.Debug("submitting batch", zap.String("name", q.cfg.Name), zap.Int("batchSize", len(itemsToSubmit)))
			err := q.batchSubmitter.Submit(itemsToSubmit)
			if err != nil {
				q.logger.Error("error submitting batch", zap.Error(err))
				q.t.SubmitErrors.With("name", q.cfg.Name).Add(1)
			}
		}()
		q.mutex.Lock()
		*items = []T{}
		q.mutex.Unlock()
	}
}

func (q *MyQueue[T]) submitIfQueueIsEmpty() bool {
	if q.cfg.SubmitOnEmptyQueue {
		if len(q.items) == 0 {
			return true
		}
	}
	return false
}
