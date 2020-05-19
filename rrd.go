/*

Copyright 2020 Andrew Hodel
	andrewhodel@gmail.com

LICENSE MIT

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.



USAGE

var rrd Rrd

24 hours with 5 minute interval
Update(5*60, 24*60/5, 'GAUGE', 34, &rrd);

30 days with 1 hour interval
Update(60*60, 30*24/1, 'GAUGE', 34, &rrd);

365 days with 1 day interval
Update(24*60*60, 365*24/1, 'GAUGE', 34, &rrd);

*/

package rrd

import (
	"time"
	"fmt"
	"strconv"
)

const (
	// color codes
	ccRed = "\033[31m"
	ccBlue = "\033[34m"
	ccReset = "\033[0m"
)

type Rrd struct {
	D			[][]float64	`json:"d"`
	R			[][]float64	`json:"r"`
	CurrentStep		int64		`json:"currentStep"`
	CurrentAvgCount		int64		`json:"currentAvgCount"`
	// use a pointer for FirstUpdateTs so we can check for nil
	FirstUpdateTs		*int64		`json:"firstUpdateTs"`
	LastUpdateDataPoint	[]float64	`json:"lastUpdateDataPoint"`
}

func Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []float64, jsonDb *Rrd) {

	if (updateDataPoint == nil) {
		return
	}

	if (jsonDb.FirstUpdateTs == nil) {
		fmt.Println("FirstUpdateTs is nil")
	}

	// get milliseconds since unix epoch
	var updateTimeStamp int64 = time.Now().Unix() * 1000

	// intervalSeconds - time between updates
	// totalSteps - total steps of data
	// dataType - GAUGE or COUNTER
	//  GAUGE - things that have no limits, like the value of raw materials
	//  COUNTER - things that count up, if we get a value that's less than last time it means it reset... stored as a per second rate
	// updateTimeStamp - unix epoch timestamp of this update
	// updateDataPoint - data object for this update
	// jsonDb - data from previous updates
	//
	// returns json object with update added

	fmt.Println("\n" + ccRed + "### GOT NEW " + dataType + " UPDATE ###" + ccReset);
	fmt.Println("intervalSeconds: " + strconv.FormatInt(intervalSeconds, 10));
	fmt.Println("totalSteps: " + strconv.FormatInt(totalSteps, 10));
	fmt.Println("updateTimeStamp: " + strconv.FormatInt(updateTimeStamp, 10));
	fmt.Println("updateDataPoint: ");
	fmt.Println(updateDataPoint);
	fmt.Println("jsonDb: ");
	fmt.Printf("%+v\n", jsonDb)

	// store updateDataPoint array as lastUpdateDataPoint
	jsonDb.LastUpdateDataPoint = updateDataPoint;

	if (jsonDb.FirstUpdateTs != nil) {
		// if the updateTimeStamp is farther away than firstUpdateTs+(totalSteps*intervalSeconds*1000)
		// then it is an entirely new chart
		if (updateTimeStamp >= *jsonDb.FirstUpdateTs+(totalSteps*2*intervalSeconds*1000)) {
			// set firstUpdateTs to nil so this will be considered the first update
			fmt.Println(ccBlue + "### THIS UPDATE IS SO MUCH NEWER THAN THE EXISTING DATA THAT IT REPLACES IT ###" + ccReset);
			jsonDb.FirstUpdateTs = nil

			// reset all the data
			if (dataType == "COUNTER") {
				// counter types need a rate calculation
				jsonDb.R = nil
				jsonDb.R = make([][]float64, totalSteps)
			}

			jsonDb.D = nil
			jsonDb.D = make([][]float64, totalSteps)
			jsonDb.CurrentStep = 0

		}
	}

	// first we need to see if this is the first update or not
	if (jsonDb.FirstUpdateTs == nil) {
		// this is the first update
		fmt.Println(ccBlue + "### INSERTING FIRST UPDATE ###" + ccReset)

		// create the array of data points
		jsonDb.D = make([][]float64, totalSteps)
		if (dataType == "COUNTER") {
			jsonDb.R = make([][]float64, totalSteps)
		}

		// insert the data for each data point
		for e := range updateDataPoint {
			fmt.Println(updateDataPoint[e])
			jsonDb.D[0] = append(jsonDb.D[0], updateDataPoint[e])
		}

		// set the firstUpdateTs by first allocating space, then assigning the value
		jsonDb.FirstUpdateTs = new(int64)
		jsonDb.FirstUpdateTs = &updateTimeStamp

	} else {

		// this is not the first update
		fmt.Println(ccBlue + "### PROCESSING " + dataType + " UPDATE ###" + ccReset)

		// this timestamp
		fmt.Println("updateTimeStamp: " + strconv.FormatInt(updateTimeStamp, 10))

		// get the time steps for each position, based on firstUpdateTs
		var timeSteps []int64
		var currentTimeSlot int64 = 0
		var c int64 = 0
		for (c < totalSteps) {
			timeSteps = append(timeSteps, *jsonDb.FirstUpdateTs + (intervalSeconds * 1000 * c))

			if (updateTimeStamp >= *jsonDb.FirstUpdateTs + (intervalSeconds * 1000 * c)) {
				currentTimeSlot = c
			}

			c++
		}

		fmt.Println("currentTimeSlot: " + strconv.FormatInt(currentTimeSlot, 10))

		// now check if this update is in the current time slot
		if (updateTimeStamp > timeSteps[jsonDb.CurrentStep+1]) {
			// this update is in a completely new time slot
			fmt.Println(ccBlue + "##### NEW STEP ##### this update is in a new step" + ccReset)

			// set the currentStep to the currentTimeSlot
			jsonDb.CurrentStep = currentTimeSlot

			// shift the data set
			if (jsonDb.CurrentStep >= totalSteps-1) {

				// calculate how much to shift by
				var shift int64 = 1
				if (updateTimeStamp >= *jsonDb.FirstUpdateTs+(totalSteps*intervalSeconds*1000)) {
					// this update needs to shift by more than 1 but obviously not more than the entire data set length
					// because if that were true, the data would have already been reset
					var time_diff int64 = updateTimeStamp - (*jsonDb.FirstUpdateTs+(totalSteps*intervalSeconds*1000));
					// shift by the number of steps beyond the last considering the original firstUpdateTs
					shift = (time_diff / (intervalSeconds * 1000)) - 1;
				}

				if (shift > 0) {

					// shift the data set
					fmt.Println(ccRed + "FIXME shifting data set by: " + strconv.FormatInt(shift, 10) + ccReset);

/*
					// remove the first _shift_ entries
					jsonDb.d.splice(0, shift);
					// add empty entries to the end _shift_ times
					var e int64 = 0
					for (e < shift) {
						var n = [];
						for (var ee=0; ee<updateDataPoint.length; ee++) {
							n.push(null);
						}
						jsonDb.d.push(n);
						e++
					}

					if (dataType == 'COUNTER') {
						// remove the first _shift_ entries
						jsonDb.r.splice(0, shift);
						// add empty entries to the end _shift_ times
						for (var e=0; e<shift; e++) {
							var n = [];
							for (var ee=0; ee<updateDataPoint.length; ee++) {
								n.push(null);
							}
							jsonDb.r.push([]);
						}
					}

					// add intervalSeconds to firstUpdateTs
					jsonDb.firstUpdateTs = jsonDb.firstUpdateTs+(intervalSeconds*1000*shift);

					jsonDb.currentStep -= shift;
					fmt.Println(ccRed + "changed currentStep: " + jsonDb.currentStep + ccReset);
*/

				}
			}

			if (jsonDb.CurrentStep+1 == totalSteps) {
				// this is needed after a shift of more than 1 but less than totalSteps
				// in case there is an update which is beyond the last when calculated against a new firstUpdateTs that may be milliseconds beyond the previous firstUpdateTs
				jsonDb.CurrentStep--
			}

			fmt.Println(ccBlue + "inserting data at: " + strconv.FormatInt(jsonDb.CurrentStep, 10) + ccReset)

			// handle different dataType
			if (dataType == "GAUGE") {

				// insert the data for each data point
				for e := range updateDataPoint {
					jsonDb.D[jsonDb.CurrentStep][e] = updateDataPoint[e]
				}

				// set the avgCount to 1
				jsonDb.CurrentAvgCount = 1

			} else if (dataType == "COUNTER") {

				// for each data point
				for e := range updateDataPoint {

					// we need to check for overflow, overflow happens when a counter resets so we check the last values to see if they were close to the limit if the previous update
					// is 3 times the size or larger, meaning if the current update is 33% or smaller it's probably an overflow
					if (jsonDb.D[jsonDb.CurrentStep-1][e] > updateDataPoint[e]*3) {

						// oh no, the counter has overflown so we need to check if this happened near 32 or 64 bit limit
						fmt.Println(ccBlue + "overflow" + ccReset)

						// the 32 bit limit is 2,147,483,647 so we should check if we were within 10% of that either way on the last update
						if (jsonDb.D[jsonDb.CurrentStep][e]<(2147483647*.1)-2147483647) {
							// this was so close to the limit that we are going to make 32bit adjustments
							// for this calculation we just need to add the remainder of subtracting the last data point from the 32 bit limit to the updateDataPoint
							updateDataPoint[e] += 2147483647-jsonDb.D[jsonDb.CurrentStep-1][e]

							// the 64 bit limit is 9,223,372,036,854,775,807 so we should check if we were within 1% of that
						} else if (jsonDb.D[jsonDb.CurrentStep][e]<(9223372036854775807*.01)-9223372036854775807) {
							// this was so close to the limit that we are going to make 64bit adjustments
							// for this calculation we just need to add the remainder of subtracting the last data point from the 64 bit limit to the updateDataPoint
							updateDataPoint[e] += 9223372036854775807-jsonDb.D[jsonDb.CurrentStep-1][e]

						}
					}


					//if (jsonDb.D[jsonDb.CurrentStep-1][e] != nil) {
						// for a counter, we need to divide the difference of this step and the previous step by
						// the difference in seconds between the updates
						var rate float64 = updateDataPoint[e]-jsonDb.D[jsonDb.CurrentStep-1][e]
						fmt.Println("calculating the rate for " + strconv.FormatFloat(rate, 'f', -1, 64) + " units over " + strconv.FormatInt(intervalSeconds, 10) + " seconds")
						rate = rate / float64(intervalSeconds)
						fmt.Println("inserting data with rate " + strconv.FormatFloat(rate, 'f', -1, 64) + " at time slot " + strconv.FormatInt(jsonDb.CurrentStep, 10))
						//jsonDb.R[jsonDb.CurrentStep][e] = rate
						// FIXME this may need to be explicity put at the index rather than just appended
						jsonDb.R[jsonDb.CurrentStep] = append(jsonDb.R[jsonDb.CurrentStep], rate)
					//}

					// insert the data
					//jsonDb.D[jsonDb.CurrentStep][e] = updateDataPoint[e];
					// FIXME this may need to be explicity put at the index rather than just appended
					jsonDb.D[jsonDb.CurrentStep] = append(jsonDb.D[jsonDb.CurrentStep], updateDataPoint[e])

				}

			} else {
				fmt.Println("unsupported dataType " + dataType)
			}

		} else {

			// being here means that this update is in the same step group as the previous
			fmt.Println("##### SAME STEP ##### this update is in the same step as the previous");

			// handle different dataType
			if (dataType == "GAUGE") {
				// this update needs to be averaged with the last

				// we need to do this for each data point
				for e := range updateDataPoint {

					var avg float64

					if (jsonDb.CurrentAvgCount > 1) {
						// we are averaging with a previous update that was itself an average
						fmt.Println("we are averaging with a previous update that was itself an average");

						// that means we have to multiply the avgCount of the previous update by the data point of the previous update
						if (jsonDb.CurrentStep == 0) {
							// this is the first update, we need to average with currentStep not the previous step
							avg = float64(jsonDb.CurrentAvgCount) * jsonDb.D[jsonDb.CurrentStep][e]
						} else {
							avg = float64(jsonDb.CurrentAvgCount) * jsonDb.D[jsonDb.CurrentStep-1][e]
						}
						// add this updateDataPoint
						avg += updateDataPoint[e]
						// increment the avg count
						jsonDb.CurrentAvgCount++
						// then divide by the avgCount
						avg = avg/float64(jsonDb.CurrentAvgCount)

						fmt.Println("updating data point with avg " + strconv.FormatFloat(avg, 'f', -1, 64))
						jsonDb.D[jsonDb.CurrentStep][e] = avg

					} else {
						// we need to average the previous update with this one
						fmt.Println("averaging with previous update")

						// we need to add the previous update data point to this one then divide by 2 for the average
						avg = (updateDataPoint[e]+jsonDb.D[jsonDb.CurrentStep][e])/2
						// set the avgCount to 2
						jsonDb.CurrentAvgCount = 2
						// and insert it
						fmt.Println("updating data point with avg " + strconv.FormatFloat(avg, 'f', -1, 64))
						jsonDb.D[jsonDb.CurrentStep][e] = avg

					}

				}

			} else if (dataType == "COUNTER") {
				// increase the counter on the last update to this one for each data point
				// this actually means to modify, not increase because it would be an increased value
				for e := range updateDataPoint {
					jsonDb.D[jsonDb.CurrentStep][e] = updateDataPoint[e]
				}

			} else {
				fmt.Println("unsupported dataType " + dataType)

			}
		}

	}

}
