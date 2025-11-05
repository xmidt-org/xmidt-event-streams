// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"github.com/xmidt-org/xmidt-event-streams/internal/kinesis"
	"github.com/xmidt-org/xmidt-event-streams/sender"

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
	SenderProvider sender.SenderProvider
}

var SenderModule = fx.Module("sender",
	fx.Provide(
		func(in SenderIn) (SenderOut, error) {
			kinesisProvider := kinesis.NewKinesisProvider()
			var opt []opts.Option[sender.SenderFactory]
			opt = append(opt,
				sender.WithLogger(in.Logger),
				sender.WithKinesisProvider(kinesisProvider),
			)

			senderProvider, err := sender.NewSenderFactory(opt)

			return SenderOut{
				SenderProvider: senderProvider}, err
		},
	),
)
