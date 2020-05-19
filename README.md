go-rrd - a pure Go [r]ound [r]obin [d]atabase library

Simple RRD's.

Installation
======
go get github.com/andrewhodel/rrd

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
	// use a pointer for FirstUpdateTs so we can check for nil
	FirstUpdateTs		*int64		`json:"firstUpdateTs"`
	LastUpdateDataPoint	[]float64	`json:"lastUpdateDataPoint"`
}
</pre>


__rrd.Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []float64, rrdPtr *Rrd)__

Updates an Rrd struct via a pointer

* intervalSeconds		time between updates
* totalSteps			total steps of data
* dataType			GAUGE or COUNTER
<pre>
    GAUGE - things that have no limits, like the value of raw materials
    COUNTER - things that count up, if we get a value that's less than last time it means it reset... stored as a per second rate
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

License
=======

Copyright 2020 Andrew Hodel
	andrewhodel@gmail.com

LICENSE MIT

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
