go-rrd - a pure Go [r]ound [r]obin [d]atabase library

Simple RRD's.

Video of the Example
======
https://www.youtube.com/watch?v=rWf1zqOcAag

Installation
======
`GO111MODULE=off go get github.com/andrewhodel/rrd`

Run your Go program with `GO111MODULE=off go run program.go`

Example
======

On a linux system you can run example/example.go which provides a simple example of collecting and displaying
traffic statistics for an interface using a COUNTER

Documentation
=============

__rrd.Rrd__

<pre>
type Rrd struct {
	D			[][]float64	`json:"d"`
	R			[][]float64	`json:"r"`
	CurrentStep		int64		`json:"currentStep"`
	CurrentAvgCount		int64		`json:"currentAvgCount"`
	FirstUpdateTs		*int64		`json:"firstUpdateTs"`
	LastUpdateDataPoint	[]float64	`json:"lastUpdateDataPoint"`
}
</pre>


__rrd.Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []float64, rrdPtr *Rrd)__

Updates an Rrd struct via a pointer.

* intervalSeconds		time between updates
* totalSteps			total steps of data
* dataType			GAUGE or COUNTER
<pre>
    GAUGE - values that stay within the range of defined integer types, like the value of raw materials.
    COUNTER - values that count and can exceed the maximum of a defined integer type.
</pre>
* updateDataPoint[]		array of data points for the update, you must maintain the same order on following Update()'s
* rrdPtr			pointer to an rrd.Rrd struct

<pre>
//24 hours with 5 minute interval
rrd.Update(5*60, 24*60/5, 'GAUGE', []float64 {34, 100}, &rrdPtr);

//30 days with 1 hour interval
rrd.Update(60*60, 30*24, 'GAUGE', []float64 {34, 100}, &rrdPtr);

//365 days with 1 day interval
rrd.Update(24*60*60, 365*24, 'GAUGE', []float64 {34, 100}, &rrdPtr);
</pre>

__rrd.Dump(rrdPtr *Rrd)__

Print the Rrd to the screen in a readable format.

Patterns
========

The pattern gathering and detection work is in patterns/patterns.go.

License
=======

Copyright 2022 Andrew Hodel
	andrewhodel@gmail.com

LICENSE MIT

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
