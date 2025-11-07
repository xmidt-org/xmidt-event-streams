// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package filter

import (
	"regexp"
	"strings"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"

	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/xmidt-event-streams/internal/queue"
	"github.com/xmidt-org/xmidt-event-streams/internal/sender"
)

type Filter interface {
	Shutdown(bool)
	Queue(*wrp.Message)
}

type BaseFilter struct {
	id string
	//endpoints []Endpoint

	streamVersion    string
	eventMatchers    []*regexp.Regexp
	metadataMatchers []*regexp.Regexp
	deliveryRetries  int
	metrics          *FilterMetrics
	wg               sync.WaitGroup
	workers          *semaphore.Weighted
	maxWorkers       int
	batchSize        int
	queueSize        int
	logger           *zap.Logger
	mutex            sync.RWMutex
	dispatcher       Dispatcher
	queue            queue.MyQueue[*wrp.Message]
	destType         sender.DestType
}

// Shutdown causes the filter to stop its activities either gently or
// abruptly based on the gentle parameter.  If gentle is false, all queued
// messages will be dropped without an attempt to send made.
func (f *BaseFilter) Shutdown(gentle bool) {
	defer f.wg.Done()
	f.dispatcher.Shutdown(gentle)
}

func overlaps(sl1 []string, sl2 []string) bool {
	for _, s1 := range sl1 {
		for _, s2 := range sl2 {
			if s1 == s2 {
				return true
			}
		}
	}
	return false
}

func (f *BaseFilter) Queue(msg *wrp.Message) {
	events := f.eventMatchers
	meta := f.metadataMatchers

	var (
		matchEvent  bool
		matchDevice = true
	)

	for _, eventRegex := range events {
		if eventRegex.MatchString(strings.TrimPrefix(msg.Destination, "event:")) {
			matchEvent = true
			break
		}
	}

	if !matchEvent {
		f.logger.Debug("destination regex doesn't match", zap.String("event.dest", msg.Destination))
		return
	}

	if meta != nil {
		matchDevice = false
		for _, deviceRegex := range meta {
			if deviceRegex.MatchString(msg.Source) || deviceRegex.MatchString(strings.TrimPrefix(msg.Destination, "event:")) {
				matchDevice = true
				break
			}
		}
	}

	if !matchDevice {
		f.logger.Debug("device regex doesn't match", zap.String("event.source", msg.Source))
		return
	}

	f.dispatcher.Queue(msg)
}
