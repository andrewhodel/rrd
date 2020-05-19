go-rrd - a pure Go [r]ound [r]obin [d]atabase library

Simple RRD's.

Example
======

On a linux system you can run example.js which provides a simple example of collecting and displaying
your enp3s0 interface traffic statistics with a COUNTER and your free/total memory using a GAUGE.

Documentation
=============

__update(intervalSeconds, totalSteps, dataType, updateTimeStamp, updateDataPoint[], jsonDb, precision=2);__

returns a JSON object representing the RRD database.

* intervalSeconds		time between updates
* totalSteps			total steps of data
* dataType			GAUGE or COUNTER
<pre>
    GAUGE - things that have no limits, like the value of raw materials
    COUNTER - things that count up, if we get a value that's less than last time it means it reset... stored as a per second rate
</pre>
* updateTimeStamp		seconds since unix epoch, not milliseconds
* updateDataPoint[]		array of data points for the update, you must maintain the same order on following update()'s
* jsonDb			data from previous updates
* precision (optional)		number of decimal places to round to for non whole numbers, default 2

<pre>
//24 hours with 5 minute interval
update(5*60, 24*60/5, 'GAUGE', [34,100], jsonObject);

//30 days with 1 hour interval
update(60*60, 30*24/1, 'GAUGE', [34,100], jsonObject);

//365 days with 1 day interval
update(24*60*60, 365*24/1, 'GAUGE', [34,100], jsonObject);
</pre>

JSON Data Format
================

The JSON Object returned by update() looks like this:

<pre>
{ d: [Array],
  currentStep: 19,
  firstUpdateTs: 1523555609625,
  r: [Array] }
</pre>

d is an array which is the length you specified in totalSteps to update() that contains the data points

firstUpdateTs is a unix epoch timestamp in milliseconds of the first update in the series

r is an array which is the length you specified in totalSteps to update() that contains the rates for the data points

You can calculate the time of each update by adding the multiple of the value of intervalSeconds which you gave to update() for each time slot.

License
=======

Copyright 2020 Andrew Hodel
	andrewhodel@gmail.com

LICENSE MIT

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
