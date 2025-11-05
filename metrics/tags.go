// SPDX-FileCopyrightText: 2017 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package metrics

// events
const (
	EventReceived                      = "event_received"
	EventSent                          = "event_sent"
	NewRecordWritten                   = "new_record_written_to_db"
	RecordUpdated                      = "record_updated_to_db"
	EventOnlineReceived                = "event_online_received"
	EventOfflineReceived               = "event_offline_received"
	EventOperationalReceived           = "event_operational_received"
	EventManageableReceived            = "event_manageableble_received"
	EventPendingRebootReceived         = "event_pending_reboot_received"
	EventUknownDestTypeReceived        = "event_unknown_dest_type_received"
	OldEventReceived                   = "old_event_received"
	OtherConnectionInterfaceReceived   = "other_ci_received"
	UnknownConnectionInterfaceReceived = "unknown_ci_received"
	ConnectionInterfaceReceived        = "ci_received"
	FutureEventReceived                = "future_event_received"
	KinesisRetryScheduled              = "kinesis_retry_scheduled"
	KinesisBatchSent                   = "kinesis_batch_sent"
	KinesisRecordSent                  = "kinesis_record_sent"
	InterfaceUsedRead                  = "interface_used_read"
	EventNotThrottled                  = "event_not_throttled"
	EventThrottled                     = "event_throttled"
	NotAnEvent                         = "not_an_event"
	DbWrite                            = "db_write_attempt"
	DbRead                             = "db_read_attempt"
)

// errors
const (
	BootTimeParseError              = "boot_time_parse_error"
	CpeTimestampParseError          = "cpe_ts_parse_error"
	CpeMissingTimestampError        = "cpe_ts_missing_error"
	EventReadRequestError           = "event_read_request"
	EventBadRequest                 = "event_bad_request"
	EventInvalidSessionId           = "event_invalid_session_id"
	EventNoMacFound                 = "event_no_mac_found"
	EventInvalidMac                 = "event_invalid_mac"
	DbWriteError                    = "db_write_error"
	DbReadError                     = "db_read_error"
	DbWriteRetry                    = "db_write_retry"
	DbReadRetry                     = "db_read_retry"
	KinesisSendError                = "kinesis_send_error"
	NoDisconnectPayload             = "missing_disconnect_payload"
	PayloadParseError               = "payload_parse_error"
	XmidtTimestampParseError        = "xmidt_ts_parse_error"
	XmidtPayloadTimestampParseError = "xmidt_payload_ts_parse_error"
	EventMergeError                 = "error_merging_event"
	DestTypeMissing                 = "dest_type_missing"
	NoSessionStartError             = "no_session_start"
	KinesisFailedRecords            = "kinesis_batch_failed_records"
	DbMapMarshalError               = "db_map_marshal_error"
	Panic                           = "panic"
	CacheWriteError                 = "cache_write_error"
	CacheWriteItemError             = "cache_write_item_error"
	CacheGetError                   = "cache_get_error"
	CacheGetItemError               = "cache_get_item_error"
	CacheUnmarshalError             = "cache_unmarshal_error"
	CacheMarshalError               = "cache_marshal_error"
	CacheDecompressError            = "cache_decompress_error"
	CacheCompressError              = "cache_compress_error"
	CacheNormalizeKeyError          = "cache_normalize_key_err"
	CacheWriteRetry                 = "cache_write_retry"
	CacheReadRetry                  = "cache_read_retry"
	StoreGetError                   = "store_get_error"
	EmptyHistory                    = "empty_history"
)

func GetUnknownTagIfEmpty(tag string) string {
	if tag == "" {
		return "unknown"
	}
	return tag
}
