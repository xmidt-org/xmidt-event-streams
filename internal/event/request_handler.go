// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package event

import (
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/fogfish/opts"
	uuid "github.com/satori/go.uuid"

	"github.com/xmidt-org/wrp-go/v3"
)

const MAX_OUTSTANDING_DEFAULT = 10000

// Below is the struct that will implement our ServeHTTP method
type RequestHandler struct {
	logger       *zap.Logger
	eventHandler EventHandler
	//metrics            *RequestHandlerMetrics
	incomingQueueDepth int32
	maxOutstanding     int32
	now                func() time.Time
}

// options
var (
	//WithMaxOutstandingWorkers = opts.ForName[RequestHandler, int32]("maxOutstanding")
	WithLogger       = opts.ForType[RequestHandler, *zap.Logger]()
	WithEventHandler = opts.ForType[RequestHandler, EventHandler]()
)

func (h *RequestHandler) checkRequired() error {
	return opts.Required(h)
}

func New(opt []opts.Option[RequestHandler]) (*RequestHandler, error) {
	h := &RequestHandler{}
	if err := opts.Apply(h, opt); err != nil {
		return h, err
	}
	if err := h.checkRequired(); err != nil {
		return h, err
	}

	h.maxOutstanding = int32(MAX_OUTSTANDING_DEFAULT)

	h.now = time.Now

	return h, nil
}

func (h *RequestHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	//eventType := unknownEventType
	// find time difference, add to metric after function finishes
	// defer func(s time.Time) {
	// 	h.recordQueueLatencyToHistogram(s, eventType)
	// }(h.now())

	h.logger.Debug("Receiving incoming request...")

	// Determine the format based on Content-Type header
	var format wrp.Format
	var contentType string
	if len(request.Header["Content-Type"]) == 1 {
		contentType = request.Header["Content-Type"][0]
	}

	switch contentType {
	case "application/msgpack":
		format = wrp.Msgpack
	case "application/json":
		format = wrp.JSON
	default:
		//return a 415
		response.WriteHeader(http.StatusUnsupportedMediaType)
		response.Write([]byte("Invalid Content-Type header. Expected application/msgpack or application/json.\n"))
		h.logger.Debug("Invalid Content-Type header. Expected application/msgpack or application/json.", zap.String("content-type", contentType))
		return
	}

	outstanding := atomic.AddInt32(&h.incomingQueueDepth, 1)
	defer atomic.AddInt32(&h.incomingQueueDepth, -1)

	if int32(0) < h.maxOutstanding && h.maxOutstanding < outstanding {
		// return a 503
		response.WriteHeader(http.StatusServiceUnavailable)
		response.Write([]byte("Incoming queue is full.\n"))
		h.logger.Error("Incoming queue is full.\n")
		return
	}

	payload, err := io.ReadAll(request.Body)
	if err != nil {
		h.logger.Error("Unable to retrieve the request body.", zap.Error(err))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(payload) == 0 {
		h.logger.Error("Empty payload.")
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte("Empty payload.\n"))
		return
	}

	decoder := wrp.NewDecoderBytes(payload, format)
	msg := new(wrp.Message)

	err = decoder.Decode(msg)
	if err != nil || msg.MessageType() != 4 {
		// return a 400
		response.WriteHeader(http.StatusBadRequest)
		if err != nil {
			response.Write([]byte("Invalid payload format.\n"))
			h.logger.Debug("Invalid payload format.")
		} else {
			response.Write([]byte("Invalid MessageType.\n"))
			h.logger.Debug("Invalid MessageType.")
		}
		return
	}

	err = wrp.UTF8(msg)
	if err != nil {
		// return a 400
		//h.metrics.InvalidCount.Add(1.0)
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte("Strings must be UTF-8.\n"))
		h.logger.Debug("Strings must be UTF-8.")
		return
	}
	// since eventType is only used to enrich metrics and logging, remove invalid UTF-8 characters from the URL
	//eventType = strings.ToValidUTF8(msg.FindEventStringSubMatch(), "")

	h.eventHandler.HandleEvent(0, h.fixWrp(msg))

	// return a 202
	response.WriteHeader(http.StatusAccepted)
	response.Write([]byte("Request placed on to queue.\n"))

	h.logger.Debug("event passed to senders.", zap.Any("event", msg))
}

func (h *RequestHandler) fixWrp(msg *wrp.Message) *wrp.Message {
	// "Fix" the WRP if needed.
	var reason string

	// Default to "application/json" if there is no content type, otherwise
	// use the one the source specified.
	if msg.ContentType == "" {
		msg.ContentType = wrp.MimeTypeJson
		reason = emptyContentTypeReason
	}

	// Ensure there is a transaction id even if we make one up
	if msg.TransactionUUID == "" {
		msg.TransactionUUID = uuid.NewV4().String()
		if reason == "" {
			reason = emptyUUIDReason
		} else {
			reason = bothEmptyReason
		}
	}

	// if reason != "" {
	// 	h.metrics.ModifiedWRPCount.With(reasonLabel, reason).Add(1.0)
	// }

	return msg
}
