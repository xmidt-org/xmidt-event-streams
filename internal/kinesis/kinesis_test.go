// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package kinesis

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/fogfish/opts"
	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const Endpoint = "http://localhost"
const Region = "local"

var logger = zap.NewExample()
var shardId = "shardId"
var sequenceNumber = "123"

var putRecordOutput = kinesis.PutRecordOutput{
	EncryptionType: types.EncryptionTypeNone,
	ShardId:        &shardId,
	SequenceNumber: &sequenceNumber,
}

type mockKinesis struct{ mock.Mock }

func newMockKinesis() *mockKinesis { return &mockKinesis{} }

func (m *mockKinesis) PutRecords(ctx context.Context, input *kinesis.PutRecordsInput, optFuns ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error) {
	args := m.Called(ctx, input, optFuns)
	return args.Get(0).(*kinesis.PutRecordsOutput), args.Error(1)
}

func (m *mockKinesis) PutRecord(ctx context.Context, input *kinesis.PutRecordInput, optFuns ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error) {
	m.Called(ctx, input, optFuns)
	return &putRecordOutput, nil
}

func (m *mockKinesis) GetRecords(ctx context.Context, input *kinesis.GetRecordsInput, optFuns ...func(*kinesis.Options)) (*kinesis.GetRecordsOutput, error) {
	m.Called(ctx, input, optFuns)

	return nil, nil
}

func (m *mockKinesis) DescribeStream(ctx context.Context, input *kinesis.DescribeStreamInput, optFuns ...func(*kinesis.Options)) (*kinesis.DescribeStreamOutput, error) {
	m.Called(ctx, input, optFuns)

	return nil, nil
}

func (m *mockKinesis) GetShardIterator(ctx context.Context, input *kinesis.GetShardIteratorInput, optFuns ...func(*kinesis.Options)) (*kinesis.GetShardIteratorOutput, error) {
	m.Called(ctx, input, optFuns)

	return nil, nil
}

func TestNewNoAwsRole(t *testing.T) {
	assert := assert.New(t)

	creds := Credentials{}

	opts := []opts.Option[KinesisClient]{
		WithRegion(Region),
		WithEndpoint(Endpoint),
		WithCredentials(creds),
		WithLogger(logger),
	}

	k, err := New(opts)
	var kc = k.(*KinesisClient)

	assert.Nil(err)
	assert.NotNil(kc.svc)
	assert.Equal(Endpoint, kc.endpoint)
	assert.Equal(Region, kc.region)
	assert.Equal(logger, kc.logger)
	assert.Equal(creds, kc.creds)
}

func TestMissingRequiredOpt(t *testing.T) {

	opts := []opts.Option[KinesisClient]{
		WithRegion(Region),
		WithEndpoint(Endpoint),
	}

	_, err := New(opts)
	assert.NotNil(t, err)
}

func TestPutRecords(t *testing.T) {
	items := []Item{{PartitionKey: "mac1", Item: []byte("test1")}, {PartitionKey: "mac2", Item: []byte("test2")}}
	stream := "some-stream"

	failedRecordInputCount := int32(10)
	putRecordsOutput := &kinesis.PutRecordsOutput{
		EncryptionType:    types.EncryptionTypeNone,
		FailedRecordCount: &failedRecordInputCount,
	}

	var m = newMockKinesis()
	m.On("PutRecords", mock.Anything, mock.Anything, mock.Anything).Return(putRecordsOutput, nil)
	kc := KinesisClient{
		logger: logger,
		svc:    m,
	}

	failedRecordCount, err := kc.PutRecords(items, stream)
	t.Log(err)
	assert.Nil(t, err)
	assert.Equal(t, 10, failedRecordCount)
}

func TestPutRecordsError(t *testing.T) {
	items := []Item{{PartitionKey: "mac1", Item: []byte("test1")}, {PartitionKey: "mac2", Item: []byte("test2")}}
	stream := "some-stream"

	failedRecordInputCount := int32(0)
	putRecordsOutput := &kinesis.PutRecordsOutput{
		EncryptionType:    types.EncryptionTypeNone,
		FailedRecordCount: &failedRecordInputCount,
	}

	var m = newMockKinesis()
	m.On("PutRecords", mock.Anything, mock.Anything, mock.Anything).Return(putRecordsOutput, errors.New("some db error"))
	kc := KinesisClient{
		logger: logger,
		svc:    m,
	}

	failedRecordCount, err := kc.PutRecords(items, stream)
	t.Log(err)
	assert.NotNil(t, err)
	assert.Equal(t, 0, failedRecordCount)
}

func TestPutRecordsErrorNilOutput(t *testing.T) {
	items := []Item{{PartitionKey: "mac1", Item: []byte("test1")}, {PartitionKey: "mac2", Item: []byte("test2")}}
	stream := "some-stream"

	putRecordsOutput := (*kinesis.PutRecordsOutput)(nil)
	var m = newMockKinesis()
	m.On("PutRecords", mock.Anything, mock.Anything, mock.Anything).Return(putRecordsOutput, errors.New("some db error"))
	kc := KinesisClient{
		logger: logger,
		svc:    m,
	}

	failedRecordCount, err := kc.PutRecords(items, stream)
	t.Log(err)
	assert.NotNil(t, err)
	assert.Equal(t, 0, failedRecordCount)
}

func TestPutRecord(t *testing.T) {
	event := []byte("test")
	stream := "some-stream"
	partitionKey := "some-key"

	var m = newMockKinesis()
	m.On("PutRecord", mock.Anything, mock.Anything, mock.Anything).Return(&putRecordOutput)
	kc := KinesisClient{
		logger: logger,
		svc:    m,
	}

	_, err := kc.PutRecord(event, stream, partitionKey)
	t.Log(err)
	assert.Nil(t, err)
}
