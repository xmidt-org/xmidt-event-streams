// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package event

import (
	kit "github.com/go-kit/kit/metrics"
)

const (
	ErrorRequestBodyCounter         = "error_request_body_count"
	EmptyRequestBodyCounter         = "empty_request_body_count"
	ModifiedWRPCounter              = "modified_wrp_count"
	DeliveryCounter                 = "delivery_count"
	DeliveryRetryCounter            = "delivery_retry_count"
	DeliveryRetryMaxGauge           = "delivery_retry_max"
	SlowConsumerDroppedMsgCounter   = "slow_consumer_dropped_message_count"
	SlowConsumerCounter             = "slow_consumer_cut_off_count"
	IncomingQueueDepth              = "incoming_queue_depth"
	IncomingEventTypeCounter        = "incoming_event_type_count"
	DropsDueToInvalidPayload        = "drops_due_to_invalid_payload"
	OutgoingQueueDepth              = "outgoing_queue_depths"
	DropsDueToPanic                 = "drops_due_to_panic"
	ConsumerRenewalTimeGauge        = "consumer_renewal_time"
	ConsumerDeliverUntilGauge       = "consumer_deliver_until"
	ConsumerDropUntilGauge          = "consumer_drop_until"
	ConsumerDeliveryWorkersGauge    = "consumer_delivery_workers"
	ConsumerMaxDeliveryWorkersGauge = "consumer_delivery_workers_max"
	QueryDurationHistogram          = "query_duration_histogram_seconds"
	IncomingQueueLatencyHistogram   = "incoming_queue_latency_histogram_seconds"
)

type RequestHandlerMetrics struct {
	ErrorRequests        kit.Counter
	EmptyRequests        kit.Counter
	InvalidCount         kit.Counter
	IncomingQueueDepth   kit.Gauge
	ModifiedWRPCount     kit.Counter
	IncomingQueueLatency kit.Histogram
}
