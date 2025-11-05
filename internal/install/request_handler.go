// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"eventstream/internal/event"

	"github.com/fogfish/opts"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type HandlerIn struct {
	fx.In
	Logger         *zap.Logger
	RequestHandler RequestHandler
	EventHandler   event.EventHandler
	//Metrics        *event.RequestHandlerMetrics
}

type HandlerOut struct {
	fx.Out
	Handler *event.RequestHandler
}

var HandlerModule = fx.Module("request_handler",
	fx.Provide(
		func(in HandlerIn) (HandlerOut, error) {
			var opts []opts.Option[event.RequestHandler]

			eventHandlerImpl := in.EventHandler.(*event.WrpEventHandler)
			if eventHandlerImpl.Logger == nil {
				in.Logger.Warn("eventHandlerImpl.Logger is nil")
			}

			opts = append(opts,
				//event.WithMaxOutstandingWorkers(in.RequestHandler.MaxOutstanding),
				event.WithLogger(in.Logger),
				event.WithEventHandler(in.EventHandler),
			)

			h, err := event.New(opts)

			return HandlerOut{
				Handler: h}, err
		},
	),
)
