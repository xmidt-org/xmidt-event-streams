# xmidt-event-streams

[![CI](https://github.com/xmidt-org/xmidt-event-streams/actions/workflows/ci.yml/badge.svg)](https://github.com/xmidt-org/xmidt-event-streams/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/xmidt-org/xmidt-event-streams/branch/main/graph/badge.svg)](https://codecov.io/gh/xmidt-org/xmidt-event-streams)
[![Go Reference](https://pkg.go.dev/badge/github.com/xmidt-org/xmidt-event-streams.svg)](https://pkg.go.dev/github.com/xmidt-org/xmidt-event-streams)
[![Apache 2.0 License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](https://github.com/xmidt-org/xmidt-event-streams/blob/main/LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/xmidt-org/xmidt-event-streams)](https://github.com/xmidt-org/xmidt-event-streams/releases)

## What Does This Library Do?
Accepts a wrp.Message and pipes it to one or more configured stream(s) based on event type.

This library currently has a built in module for uber fx and prometheus metrics. 

## Usage
```go
import "github.com/xmidt-org/xmidt-event-streams/filter"
...
msg := &wrp.Message{}
filterManager.Queue(msg)
```

## Uber FX Injection

```go
opts := fx.Options(
  ...
  
  filter.FilterModule,
  ...
)
```

## Configuration
See [streams_only.yaml](streams_only.yaml).
