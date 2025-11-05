// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package event

import (
	"errors"

	"go.uber.org/zap"

	"eventstream/filter"

	"github.com/xmidt-org/wrp-go/v3"
)

type EventHandler interface {
	HandleEvent(workerID int, msg *wrp.Message)
}

type WrpEventHandler struct {
	FilterManager filter.FilterManager
	Logger        *zap.Logger
}

func NewWrpEventHandler(filterManager filter.FilterManager, logger *zap.Logger) (*WrpEventHandler, error) {
	if filterManager == nil {
		return nil, errors.New("filterManager is required")
	}
	if logger == nil {
		return nil, errors.New("logger is required")
	}
	return &WrpEventHandler{
		FilterManager: filterManager,
		Logger:        logger,
	}, nil
}

func (h *WrpEventHandler) HandleEvent(workerID int, msg *wrp.Message) {
	h.Logger.Info("Worker received a request, now passing to sender", zap.Int("workerId", workerID))
	h.FilterManager.Queue(msg)
}
