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

const (
	emptyContentTypeReason = "empty_content_type"
	emptyUUIDReason        = "empty_uuid"
	bothEmptyReason        = "empty_uuid_and_content_type"
	unknownEventType       = "unknown"

	// metric labels

	codeLabel   = "code"
	urlLabel    = "url"
	eventLabel  = "event"
	reasonLabel = "reason"

	// metric label values
	// dropped messages reasons
	unknown                               = "unknown"
	deadlineExceededReason                = "context_deadline_exceeded"
	contextCanceledReason                 = "context_canceled"
	addressErrReason                      = "address_error"
	parseAddrErrReason                    = "parse_address_error"
	invalidAddrReason                     = "invalid_address"
	dnsErrReason                          = "dns_error"
	hostNotFoundReason                    = "host_not_found"
	connClosedReason                      = "connection_closed"
	opErrReason                           = "op_error"
	networkErrReason                      = "unknown_network_err"
	updateRequestURLFailedReason          = "update_request_url_failed"
	connectionUnexpectedlyClosedEOFReason = "connection_unexpectedly_closed_eof"
	noErrReason                           = "no_err"

	// dropped message codes
	messageDroppedCode = "message_dropped"
)

type RequestHandlerMetrics struct {
	ErrorRequests        kit.Counter
	EmptyRequests        kit.Counter
	InvalidCount         kit.Counter
	IncomingQueueDepth   kit.Gauge
	ModifiedWRPCount     kit.Counter
	IncomingQueueLatency kit.Histogram
}
