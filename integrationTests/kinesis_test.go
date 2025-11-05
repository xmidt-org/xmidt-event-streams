// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package integrationTests

import (
	"context"
	"encoding/json"
	"eventstream/internal/kinesis"
	"fmt"
	"time"

	"github.com/xmidt-org/wrp-go/v3"
)

const ()

func getKinesisRecords(ctx context.Context, expect int, k kinesis.KinesisClientAPI, sessionId string) ([]wrp.Message, error) {
	var err error
	var out []wrp.Message
	for {
		fmt.Printf("searching kinesis for %s %d times len of out is %d\n", sessionId, expect, len(out))
		err = ctx.Err()
		if err != nil {
			fmt.Println("getKinesisRecords " + err.Error())
			break
		}

		out, err = getAllMatchingKinesisRecords(k, sessionId)
		if err == nil && expect <= len(out) {
			return out, nil
		}

		time.Sleep(100 * time.Millisecond)
	}
	return nil, err
}

func getAllMatchingKinesisRecords(k kinesis.KinesisClientAPI, sessionId string) ([]wrp.Message, error) {
	output, err := k.GetRecords(TestStream)
	if err != nil {
		return nil, err
	}

	out := make([]wrp.Message, 0, 1)

	for _, record := range output.Records {
		var msg wrp.Message

		err := json.Unmarshal(record.Data, &msg)
		if err != nil {
			return nil, err
		}

		if msg.SessionID == sessionId {
			out = append(out, msg)
		}

	}

	return out, nil
}
