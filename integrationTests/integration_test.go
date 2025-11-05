// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package integrationTests

import (
	"bytes"
	"context"
	"maps"

	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fogfish/opts"
	pp "github.com/k0kubun/pp/v3"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"

	"github.com/xmidt-org/wrp-go/v3"

	"eventstream/internal/kinesis"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const EventEndpoint = "http://localhost:8080/api/events"
const Region = "local"
const TestStream = "comcast-cl.device-status.local"
const KinesisEndpoint = "http://localhost:4567"
const AccessKey = "accessKey"
const SecretKey = "secretKey"
const Role = ""

var Creds = kinesis.Credentials{
	AccessKey: AccessKey,
	SecretKey: SecretKey,
}

const BasicUsername = "eventstream"
const BasicPassword = "eventstream-password"

// testFlags returns "" to run no tests, "all" to run all tests, "broken" to
// run only broken tests, and "working" to run only working tests.

// func testFlags() string {
// 	env := os.Getenv("INTEGRATION_TESTS_RUN")
// 	env = strings.ToLower(env)
// 	env = strings.TrimSpace(env)

// 	//env = "working" //  remove

// 	switch env {
// 	case "all":
// 	case "broken":
// 	case "":
// 	default:
// 		return "working"
// 	}

// 	return env
// }

func postWrpEvent(event wrp.Message) ([]byte, error) {
	encodedBytes := []byte{}

	encoder := wrp.NewEncoderBytes(&encodedBytes, wrp.Msgpack)
	if err := encoder.Encode(event); err != nil {
		return nil, err
	}

	return postToEndpoint(EventEndpoint, encodedBytes)
}

func postToEndpoint(url string, body []byte) ([]byte, error) {
	fmt.Printf("posting to %s body len %d\n", url, len(body))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		fmt.Print(err.Error())
		return nil, err
	}

	auth := BasicUsername + ":" + BasicPassword
	encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
	req.Header.Set("Authorization", "Basic "+encodedAuth)
	req.Header.Set("Content-Type", "application/msgpack")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return nil, err
	}
	defer res.Body.Close()

	responseData, err := io.ReadAll(res.Body)

	fmt.Printf("response code: %d\n", res.StatusCode)

	if err != nil {
		return nil, err
	}

	return responseData, nil
}

func generateMacAddress() ([]byte, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return []byte(""), err
	}
	// Set the local bit
	buf[0] |= 2
	return buf, nil
}

type item struct {
	dest      string
	meta      map[string]string
	partnerid string
}

type aTest struct {
	broken           bool
	outputExpected   bool
	expectedHttpCode int
	item             *item
	api              string
	deviceId         string
}

func runIt(t *testing.T, tc aTest) {
	fmt.Println("runIT called")
	assert := assert.New(t)
	require := require.New(t)

	// switch testFlags() {
	// case "":
	// 	t.Skip("skipping integration test")
	// case "all":
	// case "broken":
	// 	if !tc.broken {
	// 		t.Skip("skipping non-broken integration test")
	// 	}
	// default: // Including working
	// 	if tc.broken {
	// 		t.Skip("skipping broken integration test")
	// 	}
	// }

	k, err := kinesis.New(
		[]opts.Option[kinesis.KinesisClient]{
			kinesis.WithEndpoint(KinesisEndpoint),
			kinesis.WithRegion(Region),
			kinesis.WithCredentials(Creds),
			kinesis.WithLogger(zap.NewExample()),
			kinesis.WithRole(Role),
		},
	)
	require.NoError(err)

	mac, err := generateMacAddress()
	require.NoError(err)
	now := time.Now().UTC()

	ksession, err := ksuid.NewRandomWithTime(now)
	require.NoError(err)
	sessionID := ksession.String()

	if testing.Verbose() {
		fmt.Println(fmt.Println(ksession.Time().Unix()))
		fmt.Println("sessionId" + sessionID)
	}

	eventTS := now.UTC().Format(time.RFC3339Nano)
	pp.Println(eventTS)
	if tc.item != nil {
		payload := fmt.Sprintf("{\"ts\":\"%s\",\"reason-for-closure\":\"%s\"}", eventTS, "tired")
		pp.Println(payload)

		event := wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      fmt.Sprintf("mac:%x", mac),
			PartnerIDs:  []string{"comcast"},
			SessionID:   sessionID,
			Destination: tc.item.dest,
			Metadata: map[string]string{
				"/hw-model":              "QA1682",
				"/hw-last-reboot-reason": "I felt like it",
				"/fw-name":               "2.364s2",
				"/random-value":          "random",
				"/xmidt-timestamp":       eventTS,
			},
			Payload: []byte(payload),
		}

		if tc.item.meta != nil {
			maps.Copy(event.Metadata, tc.item.meta)
		}
		if tc.item.partnerid != "" {
			event.PartnerIDs = []string{tc.item.partnerid}
		}

		_, err = postWrpEvent(event)
		pp.Println(event)

		require.NoError(err)
	}

	max, err := time.ParseDuration(envDefault("INTEGRATION_TEST_MAXWAIT", "30s"))
	require.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), max)
	defer cancel()

	time.Sleep(500 * time.Millisecond)

	writtenEvents, err := getKinesisRecords(ctx, 1, k, sessionID)
	assert.NoError(err)
	if tc.outputExpected {
		require.NoError(err)
		assert.NotEmpty(writtenEvents)
		require.NotNil(writtenEvents[0])
		assert.Equal(sessionID, writtenEvents[0].SessionID)
		_, _ = pp.Print(writtenEvents[0])
	} else {
		require.Empty(writtenEvents)
	}
}

func envDefault(name, def string) string {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return def
	}
	return s
}
