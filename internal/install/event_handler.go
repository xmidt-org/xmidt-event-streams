// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"eventstream/filter"
	"eventstream/internal/event"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type EventHandlerIn struct {
	fx.In
	Logger        *zap.Logger
	FilterManager filter.FilterManager
}

type EventHandlerOut struct {
	fx.Out
	Handler event.EventHandler
}

var EventHandlerModule = fx.Module("event_handler",
	fx.Provide(
		func(in EventHandlerIn) (EventHandlerOut, error) {
			fmt.Println("Creating new WrpEventHandler")
			if (in.Logger) == nil {
				fmt.Println("in.Logger is nil")
			}
			h, err := event.NewWrpEventHandler(in.FilterManager, in.Logger)
			if err != nil {
				return EventHandlerOut{}, err
			}

			return EventHandlerOut{
				Handler: h,
			}, nil
		},
	),
)
