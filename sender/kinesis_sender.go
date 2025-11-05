// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"context"
	"encoding/json"
	"eventstream/internal/kinesis"
	"fmt"
	"time"

	"github.com/fogfish/opts"
	"github.com/xmidt-org/retry"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
)

const SchemaVersion = "1.0"
const Retries = 3

type KinesisSender struct {
	kc            kinesis.KinesisClientAPI
	schemaVersion string
	logger        *zap.Logger
	retries       int
	putRunner     retry.Runner[int]
}

func NewKinesisSender(retries int, config map[string]string, logger *zap.Logger, kinesisProvider kinesis.KinesisProvider) (Sender, error) {
	fmt.Println("Creating KinesisSender with config:", config)

	if len(config) == 0 {
		return nil, fmt.Errorf("config cannot be empty")
	}

	version := config[Version]
	if version == "" {
		version = SchemaVersion
	}

	if retries <= 0 {
		retries = Retries
	}

	putRunner, err := retry.NewRunner(
		retry.WithPolicyFactory[int](retry.Config{
			Interval:   10 * time.Millisecond,
			MaxRetries: retries,
		}),
	)

	if err != nil {
		return nil, err
	}

	endpoint := config[Endpoint]
	region := config[Region]
	role := config[Role]
	accessKey := config[AccessKey]
	secretKey := config[SecretKey]
	sessionToken := config[SessionToken]

	creds := kinesis.Credentials{
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
	}

	logger.Debug("creds are", zap.Any("creds", creds))

	kc, err := kinesisProvider.Get(
		[]opts.Option[kinesis.KinesisClient]{
			kinesis.WithEndpoint(endpoint),
			kinesis.WithRegion(region),
			kinesis.WithCredentials(creds),
			kinesis.WithLogger(logger),
			kinesis.WithRole(role),
		},
	)

	return &KinesisSender{
		kc:            kc,
		schemaVersion: version,
		logger:        logger,
		retries:       retries,
		putRunner:     putRunner,
	}, err
}

func (s *KinesisSender) OnEvent(msgs []*wrp.Message, url string) (int, error) {
	items := []kinesis.Item{}
	for _, m := range msgs {
		data, err := json.Marshal(m)
		if err != nil {
			s.logger.Error("error marshaling statusEvent", zap.Any("event", m), zap.Error(err))
			continue
		}
		items = append(items, kinesis.Item{Item: data, PartitionKey: m.SessionID})
	}

	attempts := 0
	failedRecordCount, err := s.putRunner.Run(
		context.Background(),
		func(_ context.Context) (int, error) {
			attempts++
			failedRecordCount, err := s.kc.PutRecords(items, url)
			if err != nil {
				s.logger.Error("kinesis.PutRecords error", zap.Int("attempt", attempts), zap.Error(err))
			}

			return failedRecordCount, err
		},
	)

	return failedRecordCount, err
}
