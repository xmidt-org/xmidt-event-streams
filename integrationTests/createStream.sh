#!/bin/bash

# SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
# SPDX-License-Identifier: Apache-2.0

export AWS_ACCESS_KEY_ID=accessKey
export AWS_SECRET_ACCESS_KEY=secretKey
export AWS_REGION=local

aws kinesis create-stream --endpoint-url=http://localhost:4567 --stream-name comcast-cl.device-status.local --shard-count 1
