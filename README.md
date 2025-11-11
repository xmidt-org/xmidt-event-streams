# What Does This Library Do?
Accepts a wrp.Message and pipes it to one or more a configured kinesis stream(s) based on event type. 

# Usage
```
import "github.com/xmidt-org/xmidt-event-streams/filter"
...
msg := &wrp.Message{}
filterManager.Queue(msg)
```

# Uber FX Injection

```
opts := fx.Options(
  ...
  
  filter.FilterModule,
  ...
)
```

# Configuration
see ./streams_only.yaml
