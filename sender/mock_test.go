// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"github.com/xmidt-org/xmidt-event-streams/internal/kinesis"

	awsKinesis "github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/fogfish/opts"
	"github.com/stretchr/testify/mock"
)

type MockKinesisProvider struct {
	mock.Mock
}

func (m *MockKinesisProvider) Get(opt []opts.Option[kinesis.KinesisClient]) (kinesis.KinesisClientAPI, error) {
	args := m.Called(opt)
	return args.Get(0).(kinesis.KinesisClientAPI), args.Error(1)
}

type MockKinesisClientAPI struct {
	mock.Mock
}

func (m *MockKinesisClientAPI) PutRecord(event []byte, stream string, partitionKey string) (*awsKinesis.PutRecordOutput, error) {
	args := m.Called(event, stream, partitionKey)
	return args.Get(0).(*awsKinesis.PutRecordOutput), args.Error(1)
}

func (m *MockKinesisClientAPI) PutRecords(items []kinesis.Item, stream string) (int, error) {
	args := m.Called(items, stream)
	return args.Int(0), args.Error(1)
}

func (m *MockKinesisClientAPI) GetRecords(stream string) (*awsKinesis.GetRecordsOutput, error) {
	args := m.Called(stream)
	return args.Get(0).(*awsKinesis.GetRecordsOutput), args.Error(1)
}
