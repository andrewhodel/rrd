**rrd** - a [r]ound [r]obin [d]ata library

RRD is round robin data, also known as time series data.

# Installation

Works on all OS with Golang support.

Install the module with `go get github.com/andrewhodel/rrd`.

# Examples

1. `example/example.go` shows network interface traffic statistics (requires unix `ifconfig`).  https://www.youtube.com/watch?v=rWf1zqOcAag
2. `example/interpolation.go` shows how to get a rate value from a network interface range based on the size of an input buffer and based on the current IO rate of a disk.
3. `example/token_queue_disk_write.go` shows how to use `rrd.TokenQueue` to limit write speed to a disk.

# Documentation

__rrd.Rrd__

`Interval`, `TotalSteps` and `DataType` must be set.

`DataType` must be:

1. `rrd.Counter` used for counters that increase or stay the same every update, supports `rollover, overflow and reset`.
2. `rrd.Gauge` used for measurements that are within a known range.

```go
type Rrd struct {
	D					[][]*float64	`xyzdb:"D" bson:"D" json:"D"`
	R					[][]*float64	`xyzdb:"R" bson:"R" json:"R"`
	CurrentAvgCount		int64			`xyzdb:"CurrentAvgCount" bson:"CurrentAvgCount" json:"CurrentAvgCount"`
	FirstUpdateTs		*time.Time		`xyzdb:"FirstUpdateTs" bson:"FirstUpdateTs" json:"FirstUpdateTs"`
	LastUpdateDataPoint	[]*float64		`xyzdb:"LastUpdateDataPoint" bson:"LastUpdateDataPoint" json:"LastUpdateDataPoint"`
	LastUpdate			time.Time		`xyzdb:"LastUpdate" bson:"LastUpdate" json:"LastUpdate"`
	MinimumDataPoints	uint64			`xyzdb:"MinimumDataPoints" bson:"MinimumDataPoints" json:"MinimumDataPoints"`
	Interval			time.Duration	`xyzdb:"Interval" bson:"Interval" json:"Interval"`
	TotalSteps			uint64			`xyzdb:"TotalSteps" bson:"TotalSteps" json:"TotalSteps"`
	DataType			uint8			`xyzdb:"DataType" bson:"DataType" json:"DataType"`
	Debug				bool			`json:"-"`
}
```

__rrd.Update(updateDataPoint []*float64, rrdPtr *Rrd)__

Updates an Rrd struct via a pointer.

```go
// 24 hours with 5 minute interval (24 * 60 / 5 samples)
var rrd_5m rrd.Rrd
rrd_5m.Interval = time.Minute * 5
rrd_5m.TotalSteps = 24 * 60 / 5
rrd_5m.DataType = rrd.Gauge

rrd.Update(rrd.GetUpdateValues(434, 700), &rrd_5m)
```

```go
// 30 days with 1 hour interval (30 * 24 samples)
var rrd_1h rrd.Rrd
rrd_1h.Interval = time.Hour
rrd_1hTotalSteps = 30 * 24
rrd_1h.DataType = rrd.Gauge

rrd.Update(rrd.GetUpdateValues(434, 700), &rrd_1h)
```

```go
// 365 days with 1 day interval (365 samples)
var rrd_1d rrd.Rrd
rrd_1d.Interval = time.Hour * 24
rrd_1dTotalSteps = 365
rrd_1d.DataType = rrd.Gauge

rrd.Update(rrd.GetUpdateValues(434, 700), &rrd_1d)
```

`rrd.Avg` can accept any number of `*rrd.Rrd` to result a stable rate.

```go
// 5 seconds with a 1 second interval (5 samples)
var rrd_short Rrd
rrd_short.Interval = time.Second
rrd_short.TotalSteps = 5
rrd_short.DataType = rrd.Counter

// 60 seconds with a 1 second interval (60 samples)
var rrd_long Rrd
rrd_long.Interval = time.Second
rrd_long.TotalSteps = 60
rrd_long.DataType = rrd.Counter

rrd.Update(rrd.GetUpdateValues(40), &rrd_short)
rrd.Update(rrd.GetUpdateValues(40), &rrd_long)

// get average of all rrd
var avg = rrd.Avg(0, &rrd_short, &rrd_long)
```

__rrd.Dump(rrdPtr *Rrd)__

Print the Rrd to the console in a readable format.

## Mutex

Writes to `rrd.Rrd` must be one at a time, reads can be concurrent.

Use `sync.RWMutex` to write to the rrd with `rrd.Update` in `Lock` and read in `RLock`.

`DebugMutex` from https://gist.github.com/andrewhodel/ed7625a14eb87404cafd37493849d1ba is helpful.

# Interpolation

Interpolation is used to map a number of a range to another range based on a pre-defined curve.

__rrd.Interpolation__

```go
type Interpolation struct {
		// at least 2 values that are sequential, regardless of direction
		// each value is expected to be of an interval with an equal duration
		//	BaseRange 0,100			Input 50			=OutputRatio .5
		//	BaseRange 0,20,100		Input 50			=OutputRatio .6875
		BaseRange					[]float64
		// exactly 2 values
		OutputRange					[]float64
		Input						float64
}
```

__rrd.Interpolate(interpolations []Interpolatation) (error, float64)__

Return the output value of the interpolation that is the farthest in the `OutputRange`.

# Token Bucket Rate Limiting

`rrd.TokenQueue` is used to manage rate limiting, tokens each with a unique size allow input to be tracked with priority and maximum wait time.

**15MB/s**

```
Token size: 4.00KB              last 3s rate: 17.56MB/s           last 10s rate: 15.48MB             last 30s rate: 15.48MB/s
Token size: 4.00KB              last 3s rate: 17.56MB/s           last 10s rate: 15.48MB             last 30s rate: 15.48MB/s
Token size: 4.00KB              last 3s rate: 8.78MB/s            last 10s rate: 12.90MB             last 30s rate: 12.90MB/s
Token size: 4.00KB              last 3s rate: 11.42MB/s           last 10s rate: 18.61MB             last 30s rate: 18.61MB/s
Token size: 4.00KB              last 3s rate: 11.42MB/s           last 10s rate: 18.61MB             last 30s rate: 18.61MB/s
Token size: 4.00KB              last 3s rate: 11.42MB/s           last 10s rate: 18.61MB             last 30s rate: 18.61MB/s
Token size: 4.00KB              last 3s rate: 11.42MB/s           last 10s rate: 18.61MB             last 30s rate: 18.61MB/s
Token size: 4.00KB              last 3s rate: 22.83MB/s           last 10s rate: 15.95MB             last 30s rate: 15.95MB/s
Token size: 4.00KB              last 3s rate: 22.83MB/s           last 10s rate: 15.95MB             last 30s rate: 15.95MB/s
```

Each `rrd.Token` has a size and the cumulative rate of transfer/transmit set in `rrd.TokenQueue` is applied to all `rrd.Token`.

```go
type TokenQueue struct {
	// the sum of each Token.Size allowed per second
	RatePerSecond		float64
	Tokens				[]*Token
	// rrd.Working with QueueToken and UnqueueToken
	// rrd.Instant with WaitToCompleteToken
	Type				uint8
	RrdPointer			*Rrd
	sync.RWMutex
}

type Token struct {
	// the size of this token, write size in bytes for example
	Size				uint64
	Pending				bool
	// the lowest Prio proceeds first
	// this allows allocation and order to be handled conceptually at each Token
	Prio				uint64
	Time				time.Time
}
```

## TokenQueue.Type `rrd.Working`

This is used to read/write to disk or with extremely large RAM writes that require time.

It can be understood as something is arriving and there is work to do because of it and the time spent doing the work matters of the calculated rate.

```go
// create a rrd.TokenQueue that limits disk writing at 200MB/s
var token_queue_pointer *rrd.TokenQueue
rrd.SetTokenQueueLimiter(&token_queue_pointer, 200 * 1000 * 1000, rrd.Working)

// write 2000MB to disk from 50,000 goroutines
var count = 0
for {

    if (count == 50000) {
        break
    }

    go func() {

        var write_size uint64 = 40000

        var data_to_write = make([]byte, write_size)

        // Prio can be set, try 1 with large files and 0 with the last write of any file
        // MaxWait can be set and is used regardless of Prio
        var token_pointer = rrd.QueueToken(&token_queue_pointer, write_size, 0, nil)

        // write to disk

        rrd.UnqueueToken(token_queue_pointer, token_pointer)

    }()

    count += 1

}
```

## TokenQueue.Type `rrd.Instant`

This is used to write to a buffer that's expected to be instant like a socket buffer.

It can be best understood as something is being sent instantly to a TCP buffer that very quickly sends it to another router's inbound TCP queues.

The time spent doing the work is always much faster than the input or output rate being shaped and `rrd.Instant` requires less processing.

```go
// create a rrd.TokenQueue that limits data sent to a TCP socket at 400 KB/s
var token_queue_pointer *rrd.TokenQueue
rrd.SetTokenQueueLimiter(&token_queue_pointer, 400 * 1000, rrd.Instant)

for {

    // send 2MB each iteration
    var send_size uint64 = 1000 * 1000 * 2

    var data_to_send = make([]byte, send_size)

    rrd.WaitToken(token_queue_pointer, write_size, 0, nil)

    // this is `instant` enough to work with rrd.Instant
    socket.Write(data_to_send)

}

```

Pattern Matching
========

Read `patterns/patterns.go`.

License
=======

Copyright 2026 Andrew Hodel
	andrewhodel@gmail.com
	andrew@xyzbots.com

LICENSE MIT

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
