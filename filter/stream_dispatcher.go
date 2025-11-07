// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"errors"
	"github.com/xmidt-org/xmidt-event-streams/internal/queue"
	"github.com/xmidt-org/xmidt-event-streams/internal/sender"

	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
)

type StreamDispatcher struct {
	id      string
	senders []sender.Sender
	streams []Stream
	queue   queue.Queue[*wrp.Message]
	logger  *zap.Logger
	metrics *FilterMetrics
}

// this will dump messages if the buffer fills up
func (d *StreamDispatcher) Queue(msg *wrp.Message) {
	d.queue.AddItem(msg)
}

func (d *StreamDispatcher) Shutdown(gentle bool) {
	if !gentle {
		d.queue.Close()
		d.queue.Empty()
	}

	d.queue.Close()
	d.queue.Wait()
}

func (d *StreamDispatcher) Submit(msgs []*wrp.Message) error {
	defer func() {
		if r := recover(); nil != r {
			d.metrics.DroppedMessage.With(urlLabel, d.id, reasonLabel, "panic").Add(float64(len(msgs)))
			d.logger.Error("stream goroutine send() panicked", zap.String("id", d.id), zap.Any("panic", r))
			// don't silence the panic
			panic(r)
		}
	}()

	if len(d.streams) == 0 {
		d.logger.Error("no streams available for submission", zap.String("id", d.id))
		d.metrics.DroppedMessage.With(urlLabel, d.id, reasonLabel, "no_streams").Add(float64(len(msgs)))
		return errors.New("no streams available for submission")
	}

	d.logger.Debug("stream dispatcher submitting batch", zap.String("stream", d.streams[0].StreamName), zap.Int("numMessages", len(msgs)))
	if len(msgs) == 0 {
		d.logger.Debug("no messages to send")
		return nil
	}

	var err error = nil
	i := 0
	s := d.streams[0].StreamName

	for {
		if i == len(d.streams) {
			// All streams have been tried
			d.logger.Error("all streams failed", zap.String("stream", s), zap.Int("attempt", i))
			d.metrics.DroppedMessage.With(urlLabel, s, reasonLabel, "all_failed").Add(float64(len(msgs)))
			break
		}
		s = d.streams[i].StreamName
		d.logger.Debug("attempting to send to stream", zap.String("stream", s), zap.Int("attempt", i), zap.Any("sender", d.senders[i]))
		err = d.submitBatch(s, d.senders[i], msgs)
		if err == nil {
			break
		}
		i++
	}

	return err
}

func (d *StreamDispatcher) submitBatch(stream string, sender sender.Sender, msgs []*wrp.Message) error {
	var err error = nil

	failedRecordCount, err := sender.OnEvent(msgs, stream)

	reason := "no_err"
	l := d.logger
	if err != nil {
		reason = "send_error"
		d.logger.Error("error writing to stream", zap.String("stream", stream), zap.String(reasonLabel, reason), zap.Error(err))
		d.metrics.DroppedMessage.With(urlLabel, stream, reasonLabel, reason).Add(1)
	} else if failedRecordCount > 0 {
		reason = "some_records_failed"
		d.logger.Error("some records failed to write to stream", zap.String(reasonLabel, reason), zap.Int("failedRecordCount", failedRecordCount))
		d.metrics.DroppedMessage.With(urlLabel, stream, reasonLabel, reason).Add(1)
		err = errors.New("some records failed to write to stream")
	}

	l.Debug("events sent-ish", zap.String("stream", stream))

	return err
}
