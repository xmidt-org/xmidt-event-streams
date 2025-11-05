// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"eventstream/internal/kinesis"
	"fmt"
	"strings"

	"github.com/fogfish/opts"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
)

type DestType int

const (
	Kinesis DestType = iota
)

const (
	Region       = "region"
	AccessKey    = "access_key"
	SecretKey    = "secret_key"
	SessionToken = "sessiontoken"
	Endpoint     = "endpoint"
	Version      = "version"
	Role         = "role"
)

// destTypeMap maps string values to DestType constants
var destTypeMap = map[string]DestType{
	"kinesis": Kinesis,
}

func ParseDestType(s string) (DestType, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	if destType, ok := destTypeMap[normalized]; ok {
		return destType, nil
	}
	return 0, fmt.Errorf("invalid DestType: %s", s)
}

type SenderProvider interface {
	GetSender(retries int, d DestType, config map[string]string) (Sender, error)
}

type Sender interface {
	OnEvent([]*wrp.Message, string) (int, error)
}

type SenderFactory struct {
	logger          *zap.Logger
	kinesisProvider kinesis.KinesisProvider
}

// options
var (
	WithLogger          = opts.ForType[SenderFactory, *zap.Logger]()
	WithKinesisProvider = opts.ForType[SenderFactory, kinesis.KinesisProvider]()
)

func (c *SenderFactory) checkRequired() error {
	return opts.Required(c,
		WithLogger(nil),
		//WithKinesisProvider(kinesis.KinesisProvider(nil)),
	)
}

func NewSenderFactory(opt []opts.Option[SenderFactory]) (SenderProvider, error) {
	sf := &SenderFactory{}

	if err := opts.Apply(sf, opt); err != nil {
		return nil, err
	}

	if err := sf.checkRequired(); err != nil {
		return nil, err
	}

	return sf, nil
}

func (sf *SenderFactory) GetSender(retries int, d DestType, config map[string]string) (Sender, error) {
	if len(config) == 0 {
		return nil, fmt.Errorf("no config provided for sender")
	}
	if d == Kinesis {
		return sf.getKinesisSender(retries, config)
	}
	return nil, fmt.Errorf("no sender for dest type %v", d)
}

func (sf *SenderFactory) getKinesisSender(retries int, config map[string]string) (Sender, error) {
	return NewKinesisSender(retries, config, sf.logger, sf.kinesisProvider)
}
