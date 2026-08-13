go-rrd - a [r]ound [r]obin [d]atabase library

# Installation
`go get github.com/andrewhodel/rrd`

Documentation
=============

# Rrd

__rrd.Rrd__

```go
type Rrd struct {
	D								[][]*float64		`json:"d"`
	R								[][]*float64		`json:"r"`
	CurrentStep						int64				`json:"currentStep"`
	CurrentAvgCount					int64				`json:"currentAvgCount"`
	FirstUpdateTs					*int64				`json:"firstUpdateTs"`
	LastUpdateDataPoint				[]*float64			`json:"lastUpdateDataPoint"`
}
```

__rrd.Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []*float64, rrdPtr *Rrd)__

Updates an Rrd struct via a pointer.

```
debug								bool				output debug to console
interval							int64				ideal time between updates
totalSteps							int64				total steps of data
dataType							uint8				rrd.Gauge or rrd.Counter
														Gauge - values that stay within the range of defined integer types, like the value of raw materials.
														Counter - values that count and can exceed the maximum of a defined integer type.
updateDataPoint						[]*float64			array of data points for the update, must have the same order in following `Update`s.  Make sure to use `rrd.GetUpdateValues` to support nil values and pointer value copying.
rrdPtr								*Rrd				pointer to an rrd.Rrd struct
```

```go
var rrdPtr Rrd

// 24 hours with 5 minute interval (24 * 60 / 5 samples)
rrd.Update(false, time.Minute * 5, 24*60/5, rrd.Gauge, rrd.GetUpdateValues(434, 700), &rrdPtr)
```

```go
var rrdPtr Rrd

// 30 days with 1 hour interval (30 * 24 samples)
rrd.Update(false, time.Hour, 30*24, rrd.Gauge, rrd.GetUpdateValues(434, 700), &rrdPtr)
```

```go
var rrdPtr Rrd

// 365 days with 1 day interval (365 samples)
rrd.Update(false, time.Hour * 24, 365, rrd.Gauge, rrd.GetUpdateValues(434, 700), &rrdPtr)
```

```go
var rrdPtr Rrd

// 5 seconds with a 1 second interval (5 samples)
rrd.Update(false, time.Second, 5, rrd.Counter, []float64 rrd.GetUpdateValues(40), &rrdPtr)

// get average of all data
// try it with /proc/diskstats field 8 (writes completed)
// this is the rate per second of writes completed averaged through the last 5 seconds
var avg = rrd.Avg(&rrdPtr, 0)
```

__rrd.Dump(rrdPtr *Rrd)__

Print the Rrd to the screen in a readable format.

## Example

On a linux system you can run `example/example.go` to show network interface traffic statistics.

## Video of the Example

https://www.youtube.com/watch?v=rWf1zqOcAag

## Rate Interval

The rates are stored as a `float64` with a 1 second interval, providing nanosecond resolution.

## Mutex

Use `sync.RWMutex` to write to the rrd with `rrd.Update` in `Lock` and read in `RLock`.

`DebugMutex` from https://gist.github.com/andrewhodel/ed7625a14eb87404cafd37493849d1ba is helpful.

# Interpolation

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

Each token has a size and the rate of transfer/transmit is applied by and to all `rrd.Token` of a `rrd.TokenQueue`.

```go
type TokenQueue struct {
	// the sum of each Token.Size allowed per second
	RatePerSecond			float64
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

## rrd.Working

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

## rrd.Instant

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

## Example

`example/interpolation.go` shows how to know the desired rate of a network interface based on the size of an input buffer and the disk activity.

Pattern/Sequence/Routine/Design/Order/Arrangement/Model/Structure Matching
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
