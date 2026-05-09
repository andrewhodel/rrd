go-rrd - a pure Go [r]ound [r]obin [d]atabase library

Simple RRD's.

Video of the Example
======
https://www.youtube.com/watch?v=rWf1zqOcAag

Installation
======
`go get github.com/andrewhodel/rrd`

Example
======

On a linux system you can run example/example.go which provides a simple example that shows network interface traffic statistics.

Documentation
=============

__rrd.Rrd__

```go
type Rrd struct {
	D			[][]float64	`json:"d"`
	R			[][]float64	`json:"r"`
	CurrentStep		int64		`json:"currentStep"`
	CurrentAvgCount		int64		`json:"currentAvgCount"`
	FirstUpdateTs		*int64		`json:"firstUpdateTs"`
	LastUpdateDataPoint	[]float64	`json:"lastUpdateDataPoint"`
}
```

__rrd.Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []float64, rrdPtr *Rrd)__

Updates an Rrd struct via a pointer.

```go
debug					bool			output debug to console
interval				int64			ideal time between updates
totalSteps				int64			total steps of data
dataType				string			GAUGE or COUNTER
	GAUGE - values that stay within the range of defined integer types, like the value of raw materials.
	COUNTER - values that count and can exceed the maximum of a defined integer type.
updateDataPoint			[]float64		array of data points for the update, must have the same order in following `Update`s.
rrdPtr					*Rrd			pointer to an rrd.Rrd struct
```

```go
var rrdPtr Rrd

// 24 hours with 5 minute interval (24 * 60 / 5 samples)
rrd.Update(false, time.Minute * 5, 24*60/5, "GAUGE", []float64 {434, 700}, &rrdPtr)
```

```go
var rrdPtr Rrd

// 30 days with 1 hour interval (30 * 24 samples)
rrd.Update(false, time.Hour, 30*24, "GAUGE", []float64 {434, 700}, &rrdPtr)
```

```go
var rrdPtr Rrd

// 365 days with 1 day interval (365 samples)
rrd.Update(false, time.Hour * 24, 365, "GAUGE", []float64 {434, 700}, &rrdPtr)
```

```go
var rrdPtr Rrd

// 5 seconds with a 1 second interval (5 samples)
rrd.Update(false, time.Second, 5, "COUNTER", []float64 {40}, &rrdPtr)

// get average of all data
// try it with /proc/diskstats field 8 (writes completed)
// this is the rate per second of writes completed averaged through the last 5 seconds
var avg = rrd.Avg(&rrdPtr, 0)
```

# Rate Interval

The rates are stored as a `float64` with a 1 second interval, providing nanosecond resolution.

__rrd.Dump(rrdPtr *Rrd)__

Print the Rrd to the screen in a readable format.

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
