// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"github.com/xmidt-org/xmidt-event-streams/internal/kinesis"

	"github.com/fogfish/opts"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SenderIn struct {
	fx.In
	Logger *zap.Logger
}

type SenderOut struct {
	fx.Out
	SenderProvider SenderProvider
}

var SenderModule = fx.Module("sender",
	fx.Provide(
		func(in SenderIn) (SenderOut, error) {
			kinesisProvider := kinesis.NewKinesisProvider()
			var opt []opts.Option[SenderFactory]
			opt = append(opt,
				WithLogger(in.Logger),
				WithKinesisProvider(kinesisProvider),
			)

			senderProvider, err := NewSenderFactory(opt)

			return SenderOut{
				SenderProvider: senderProvider}, err
		},
	),
)
