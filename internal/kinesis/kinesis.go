// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package kinesis

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"

	"eventstream/internal/batch" // Ensure this package does not import `kinesis`
	"eventstream/internal/log"   // Ensure this package does not import `kinesis`

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fogfish/opts"
	"go.uber.org/zap"
)

const MaxPutRecordsBatchSize = 500

type KinesisProvider interface {
	Get(opt []opts.Option[KinesisClient]) (KinesisClientAPI, error)
}

type KinesisFactory struct{}

func NewKinesisProvider() KinesisProvider {
	return &KinesisFactory{}
}

func (kf *KinesisFactory) Get(opt []opts.Option[KinesisClient]) (KinesisClientAPI, error) {
	return New(opt)
}

type KinesisClientAPI interface {
	PutRecord(event []byte, stream string, partitionKey string) (*kinesis.PutRecordOutput, error)
	PutRecords(items []Item, stream string) (int, error)
	GetRecords(stream string) (*kinesis.GetRecordsOutput, error)
}

type KinesisAPI interface {
	PutRecord(context.Context, *kinesis.PutRecordInput, ...func(*kinesis.Options)) (*kinesis.PutRecordOutput, error)
	PutRecords(context.Context, *kinesis.PutRecordsInput, ...func(*kinesis.Options)) (*kinesis.PutRecordsOutput, error)
	GetRecords(context.Context, *kinesis.GetRecordsInput, ...func(*kinesis.Options)) (*kinesis.GetRecordsOutput, error)
	GetShardIterator(context.Context, *kinesis.GetShardIteratorInput, ...func(*kinesis.Options)) (*kinesis.GetShardIteratorOutput, error)
	DescribeStream(context.Context, *kinesis.DescribeStreamInput, ...func(*kinesis.Options)) (*kinesis.DescribeStreamOutput, error)
}

type KinesisClient struct {
	svc           KinesisAPI
	logger        *zap.Logger
	region        string
	endpoint      string
	role          string
	creds         Credentials
	credsExpireAt time.Time
}

type Credentials struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
}

type Item struct {
	PartitionKey string
	Item         []byte
}

// options
var WithRegion = opts.ForName[KinesisClient, string]("region")
var WithEndpoint = opts.ForName[KinesisClient, string]("endpoint")
var WithRole = opts.ForName[KinesisClient, string]("role")
var WithCredentials = opts.ForType[KinesisClient, Credentials]()
var WithLogger = opts.ForType[KinesisClient, *zap.Logger]()

func (c *KinesisClient) checkRequired() error {
	return opts.Required(c,
		WithRegion(""),
		WithEndpoint(""),
		WithLogger(nil),
	)
}

func New(opt []opts.Option[KinesisClient]) (KinesisClientAPI, error) {
	c := &KinesisClient{}

	if err := opts.Apply(c, opt); err != nil {
		return nil, err
	}

	if err := c.checkRequired(); err != nil {
		return nil, err
	}

	k, credsExpireAt, err := c.getClient()
	if err != nil {
		return nil, err
	}

	c.credsExpireAt = credsExpireAt
	c.svc = k

	return c, nil
}

func (c *KinesisClient) getClient() (KinesisAPI, time.Time, error) {
	ctx := context.Background()

	var credsExpireAt time.Time
	var awsConfig aws.Config
	var err error

	c.logger.Debug("KinesisClient is ", zap.Any("kinesisClient", c))
	c.logger.Debug("kinesis role is", zap.String("role", c.role))

	if c.role != "" { // we need to authenticate with aws and assume a cross-account role

		// config to retrieve sts credentials
		awsConfig, err = config.LoadDefaultConfig(
			ctx,
			config.WithRegion(c.region),
			config.WithRetryMaxAttempts(3),
			//config.WithCredentialsChainVerboseErrors(true),
			config.WithHTTPClient(
				&http.Client{Timeout: 10 * time.Second},
			),
		)
		if err != nil {
			c.logger.Error("unable to load aws config for sts credentials", zap.String(log.Role, c.role), zap.Error(err))
			return nil, time.Time{}, err
		}

		// retrieve credentials for cross-account role
		client := sts.NewFromConfig(awsConfig)
		creds := stscreds.NewAssumeRoleProvider(client, c.role)
		stsResponse, err := creds.Retrieve(ctx)
		if err != nil {
			c.logger.Error("unable to authenticate kinesis assumed role", zap.String(log.Role, c.role), zap.Error(err))
			return nil, time.Time{}, err
		}

		// create config for kinesis client with cross account credentials
		awsConfig, err = config.LoadDefaultConfig(
			ctx,
			config.WithRegion(c.region),
			config.WithBaseEndpoint(c.endpoint),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				stsResponse.AccessKeyID,
				stsResponse.SecretAccessKey,
				stsResponse.SessionToken),
			),
		)
		if err != nil {
			c.logger.Error("unable to create kinesis client config", zap.String(log.Role, c.role), zap.Error(err))
			return nil, time.Time{}, err
		}

		credsExpireAt = stsResponse.Expires
	} else {
		// create config to talk to local kinesis
		c.logger.Debug("creating local kinesis client without role", zap.Any("creds", c.creds))
		var err error
		awsConfig, err = config.LoadDefaultConfig(
			ctx,
			config.WithRegion(c.region),
			config.WithBaseEndpoint(c.endpoint),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				c.creds.AccessKey,
				c.creds.SecretKey,
				c.creds.SessionToken,
			),
			),
		)
		if err != nil {
			c.logger.Error("unable to create local kinesis client config", zap.String(log.Role, c.role), zap.Error(err))
			return nil, time.Time{}, err
		}

	}

	k := kinesis.NewFromConfig(awsConfig)

	return k, credsExpireAt, nil
}

func (c *KinesisClient) PutRecord(event []byte, stream string, partitionKey string) (*kinesis.PutRecordOutput, error) {
	c.logger.Debug("putting event to kinesis", zap.String("partitionKey", partitionKey), zap.String(log.EventBody, string(event)))

	c.refreshClient()

	putOutput, err := c.svc.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         event,
		StreamName:   &stream,
		PartitionKey: aws.String(partitionKey),
	})

	if err != nil {
		c.logger.Error("PutRecord failed", zap.Error(err))
		return putOutput, err
	}

	c.logger.Debug("Published Event", zap.Any("result", putOutput), zap.String("endpoint", c.endpoint))
	return putOutput, nil
}

func (c *KinesisClient) PutRecords(items []Item, stream string) (int, error) {
	notDone := true
	start := 0
	failedRecordCount := int32(0)
	for notDone {
		batch, more := batch.GetBatch(start, MaxPutRecordsBatchSize, items)
		notDone = more
		start += MaxPutRecordsBatchSize

		c.logger.Debug("putting events to kinesis", zap.Int("size", len(items)))

		records := []types.PutRecordsRequestEntry{}
		for _, item := range batch {
			entry := types.PutRecordsRequestEntry{
				Data:         item.Item,
				PartitionKey: aws.String(item.PartitionKey),
			}
			records = append(records, entry)
		}

		c.refreshClient()
		putOutput, err := c.svc.PutRecords(context.Background(), &kinesis.PutRecordsInput{
			Records:    records,
			StreamName: &stream,
		})

		if putOutput != nil {
			failedRecordCount += *putOutput.FailedRecordCount
		}

		if err != nil {
			c.logger.Error("PutRecords failed", zap.Error(err))
			return int(failedRecordCount), err
		}
		c.logger.Debug("Published Records", zap.Any("result", putOutput), zap.String("endpoint", c.endpoint))
	}

	return int(failedRecordCount), nil
}

// get all records - for integration testing, only, not used in production

func (c *KinesisClient) GetRecords(stream string) (*kinesis.GetRecordsOutput, error) {
	c.logger.Debug("getting records", zap.String("endpoint", c.endpoint))

	// get starting sequence number first, then shard iterator for that
	// retrieve iterator
	shardId := "shardId-000000000000"

	description, err := c.svc.DescribeStream(context.Background(), &kinesis.DescribeStreamInput{
		StreamName: aws.String(stream),
	})

	if err != nil {
		c.logger.Error("error describing stream", zap.Error(err))
		return nil, err
	}
	if description == nil {
		return nil, fmt.Errorf("description is nil")
	}
	if description.StreamDescription == nil {
		return nil, fmt.Errorf("stream description is nil")
	}
	if len(description.StreamDescription.Shards) == 0 {
		return nil, fmt.Errorf("no shards found")
	}
	if description.StreamDescription.Shards[0].SequenceNumberRange == nil {
		return nil, fmt.Errorf("sequence number range is nil")
	}
	if description.StreamDescription.Shards[0].SequenceNumberRange.StartingSequenceNumber == nil {
		return nil, fmt.Errorf("starting sequence number is nil")
	}
	sequenceNo := description.StreamDescription.Shards[0].SequenceNumberRange.StartingSequenceNumber

	iteratorOutput, err := c.svc.GetShardIterator(context.Background(), &kinesis.GetShardIteratorInput{
		// Shard Id is provided when making put record(s) request.
		ShardId: &shardId,
		//ShardIteratorType: aws.String("TRIM_HORIZON"),
		ShardIteratorType:      types.ShardIteratorTypeAtSequenceNumber,
		StartingSequenceNumber: sequenceNo,
		// ShardIteratorType: aws.String("LATEST"),
		StreamName: &stream,
	})

	if err != nil {
		c.logger.Error("error getting iterator output", zap.Error(err))
		return nil, err
	}

	// get records use shard iterator for making request
	records, err := c.svc.GetRecords(context.Background(), &kinesis.GetRecordsInput{
		ShardIterator: iteratorOutput.ShardIterator,
		//StreamARN: &streamARN,
	})

	if err != nil {
		c.logger.Error("error getting records", zap.Error(err))
		panic(err)
	}

	if err != nil {
		c.logger.Error(fmt.Sprintf("Error: %v", err))
		return records, err
	}

	return records, nil
}

func (c *KinesisClient) refreshClient() {
	if c.role != "" {
		if c.credsExpireAt.Unix() <= time.Now().Add(3*time.Minute).Unix() {
			k, credsExpireAt, err := c.getClient()
			if err != nil {
				c.logger.Error("error refreshing kinesis client", zap.Error(err))
				return
			}

			// save expiration locally because asking for it is a drain on the metadata server
			c.credsExpireAt = credsExpireAt
			c.svc = k
		}
	}
}
