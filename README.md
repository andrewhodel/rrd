go-rrd - a [r]ound [r]obin [d]atabase library

# Installation
`go get github.com/andrewhodel/rrd`

Documentation
=============

# Rrd

__rrd.Rrd__

```go
type Rrd struct {
	D								[][]float64			`json:"d"`
	R								[][]float64			`json:"r"`
	CurrentStep						int64				`json:"currentStep"`
	CurrentAvgCount					int64				`json:"currentAvgCount"`
	FirstUpdateTs					*int64				`json:"firstUpdateTs"`
	LastUpdateDataPoint				[]float64			`json:"lastUpdateDataPoint"`
}
```

__rrd.Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []float64, rrdPtr *Rrd)__

Updates an Rrd struct via a pointer.

```
debug								bool				output debug to console
interval							int64				ideal time between updates
totalSteps							int64				total steps of data
dataType							uint8				rrd.Gauge or rrd.Counter
														Gauge - values that stay within the range of defined integer types, like the value of raw materials.
														Counter - values that count and can exceed the maximum of a defined integer type.
updateDataPoint						[]float64			array of data points for the update, must have the same order in following `Update`s.
rrdPtr								*Rrd				pointer to an rrd.Rrd struct
```

```go
var rrdPtr Rrd

// 24 hours with 5 minute interval (24 * 60 / 5 samples)
rrd.Update(false, time.Minute * 5, 24*60/5, rrd.Gauge, []float64 {434, 700}, &rrdPtr)
```

```go
var rrdPtr Rrd

// 30 days with 1 hour interval (30 * 24 samples)
rrd.Update(false, time.Hour, 30*24, rrd.Gauge, []float64 {434, 700}, &rrdPtr)
```

```go
var rrdPtr Rrd

// 365 days with 1 day interval (365 samples)
rrd.Update(false, time.Hour * 24, 365, rrd.Gauge, []float64 {434, 700}, &rrdPtr)
```

```go
var rrdPtr Rrd

// 5 seconds with a 1 second interval (5 samples)
rrd.Update(false, time.Second, 5, rrd.Counter, []float64 {40}, &rrdPtr)

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
