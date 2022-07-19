/*

Copyright 2022 Andrew Hodel
	andrewhodel@gmail.com

LICENSE MIT

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

*/

package rrd

import (
	"time"
	"fmt"
	"strconv"
	"math"
	//"math/big"
)

const (
	// color codes
	ccRed = "\033[31m"
	ccBlue = "\033[34m"
	ccReset = "\033[0m"
)

type Rrd struct {
	D			[][]float64	`bson:"d" json:"d"`
	R			[][]float64	`bson:"r" json:"r"`
	//RR			[][]big.Float	`bson:"rr" json:"rr"`
	CurrentStep		int64		`bson:"currentStep" json:"currentStep"`
	CurrentAvgCount		int64		`bson:"currentAvgCount" json:"currentAvgCount"`
	// use a pointer for FirstUpdateTs so nil values are possible
	FirstUpdateTs		*int64		`bson:"firstUpdateTs" json:"firstUpdateTs"`
	LastUpdateDataPoint	[]float64	`bson:"lastUpdateDataPoint" json:"lastUpdateDataPoint"`
	MinimumDataPoints	uint64		`bson:"minimumDataPoints" json:"minimumDataPoints"`
}

func Dump(rrdPtr *Rrd) {

	fmt.Printf("rrdPtr: CurrentStep %d, CurrentAvgCount %d, FirstUpdateTs %d\n", rrdPtr.CurrentStep, rrdPtr.CurrentAvgCount, *rrdPtr.FirstUpdateTs)
	fmt.Println("rrdPtr LastUpdateDataPoint:")

	for e := range rrdPtr.LastUpdateDataPoint {
		fmt.Printf("\t%f", rrdPtr.LastUpdateDataPoint[e])
	}

	fmt.Println("")

	fmt.Printf("rrdPtr D (COUNTER VALUES) (%d):\n", len(rrdPtr.D))

	for e := range rrdPtr.D {
		for n := range rrdPtr.D[e] {
			fmt.Printf("\tInterval %d\t%f", e, rrdPtr.D[e][n])
		}
		fmt.Println("")
	}

	fmt.Println("")

	if (rrdPtr.R != nil) {
		fmt.Printf("rrdPtr R (RATE OF COUNTER INTERVALS) (%d):\n", len(rrdPtr.R))

		for e := range rrdPtr.R {
			for n := range rrdPtr.R[e] {
				fmt.Printf("\tInterval %d\t%f", e, rrdPtr.R[e][n])
			}
			fmt.Println("")
		}

		fmt.Println("")
	}

}

func Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []float64, rrdPtr *Rrd) {
	// all timing is based on system time at execution of Update()
	// data can be sent without knowledge of time on the distant side, as if you were receiving data from an unknown source or distant planet
	// world internet latency via fiber could be more than 200ms and much longer in space

	var debug = false

	if (updateDataPoint == nil) {
		return
	}

	if (len(updateDataPoint) < int(rrdPtr.MinimumDataPoints)) {
		if debug { fmt.Printf("updateDataPoint must have at least %d values\n", rrdPtr.MinimumDataPoints) }
		return
	} else if (len(updateDataPoint) > int(rrdPtr.MinimumDataPoints)) {
		// increase the minimum length when updateDataPoint is longer
		rrdPtr.MinimumDataPoints = uint64(len(updateDataPoint))

		// make all data point arrays at least the length of this update
		for n := range rrdPtr.D {

			if (dataType == "COUNTER") {
				// R values

				if (len(rrdPtr.R[n]) == 0) {
					// skip [] values
				} else if (len(rrdPtr.R[n]) != int(rrdPtr.MinimumDataPoints)) {
					// add zeroes
					var cur_len = len(rrdPtr.R[n])
					var new_value_count = int(rrdPtr.MinimumDataPoints) - cur_len
					var l = 0
					for (l < new_value_count) {
						// add a zero for each new field
						rrdPtr.R[n] = append(rrdPtr.R[n], float64(0))
						l = l + 1
					}
				}

			}

			// D values
			if (len(rrdPtr.D[n]) == 0) {
				// skip [] values
			} else if (len(rrdPtr.D[n]) != int(rrdPtr.MinimumDataPoints)) {
				// add zeroes
				var cur_len = len(rrdPtr.D[n])
				var new_value_count = int(rrdPtr.MinimumDataPoints) - cur_len
				var l = 0
				for (l < new_value_count) {
					// add a zero for each new field
					rrdPtr.D[n] = append(rrdPtr.D[n], float64(0))
					l = l + 1
				}
			}

		}

	}

	if (rrdPtr.FirstUpdateTs == nil) {
		if debug { fmt.Println("FirstUpdateTs is nil") }
	}

	// get milliseconds since unix epoch
	var updateTimeStamp int64 = time.Now().Unix() * 1000

	// intervalSeconds - time between updates
	// totalSteps - total steps of data
	// dataType - GAUGE or COUNTER
	//  GAUGE - things that have no limits, like the value of raw materials
	//  COUNTER - things that count up, if get a value that's less than last time it means it reset... stored as a per second rate
	// updateTimeStamp - unix epoch timestamp of this update
	// updateDataPoint - data object for this update
	// rrdPtr - data from previous updates
	//
	// returns json object with update added

	if debug { fmt.Println("\n" + ccRed + "### GOT NEW " + dataType + " UPDATE ###" + ccReset) }
	if debug { fmt.Println("intervalSeconds: " + strconv.FormatInt(intervalSeconds, 10)) }
	if debug { fmt.Println("totalSteps: " + strconv.FormatInt(totalSteps, 10)) }
	if debug { fmt.Println("updateTimeStamp: " + strconv.FormatInt(updateTimeStamp, 10)) }
	if debug { fmt.Println("updateDataPoint:") }

	for e := range updateDataPoint {
		if debug { fmt.Printf("\t%f", updateDataPoint[e]) }
	}
	if debug { fmt.Println("") }

	// store updateDataPoint array as lastUpdateDataPoint
	rrdPtr.LastUpdateDataPoint = updateDataPoint

	if (rrdPtr.FirstUpdateTs != nil) {
		// if the updateTimeStamp is farther away than firstUpdateTs+(totalSteps*intervalSeconds*1000)
		// then it is an entirely new chart
		if (updateTimeStamp >= *rrdPtr.FirstUpdateTs+(totalSteps*2*intervalSeconds*1000)) {
			// set firstUpdateTs to nil, this will be considered the first update
			if debug { fmt.Println(ccBlue + "### THIS UPDATE IS SO MUCH NEWER THAN THE EXISTING DATA THAT IT REPLACES IT ###" + ccReset) }
			rrdPtr.FirstUpdateTs = nil

			// reset all the data
			if (dataType == "COUNTER") {
				// counter types need a rate calculation
				rrdPtr.R = nil
				rrdPtr.R = make([][]float64, totalSteps)
			}

			rrdPtr.D = nil
			rrdPtr.D = make([][]float64, totalSteps)
			rrdPtr.CurrentStep = 0

		}
	}

	// first need to see if this is the first update or not
	if (rrdPtr.FirstUpdateTs == nil) {
		// this is the first update
		if debug { fmt.Println(ccBlue + "### INSERTING FIRST UPDATE ###" + ccReset) }

		// create the array of data points
		rrdPtr.D = make([][]float64, totalSteps)
		if (dataType == "COUNTER") {
			rrdPtr.R = make([][]float64, totalSteps)
		}

		// insert the data for each data point
		for e := range updateDataPoint {
			if debug { fmt.Printf("\t%f", updateDataPoint[e]) }
			rrdPtr.D[0] = append(rrdPtr.D[0], updateDataPoint[e])
		}
		if debug { fmt.Println("") }

		// set the firstUpdateTs by first allocating space, then assigning the value
		rrdPtr.FirstUpdateTs = new(int64)
		rrdPtr.FirstUpdateTs = &updateTimeStamp

	} else {

		// this is not the first update
		if debug { fmt.Println(ccBlue + "### PROCESSING " + dataType + " UPDATE ###" + ccReset) }

		// this timestamp
		if debug { fmt.Println("updateTimeStamp: " + strconv.FormatInt(updateTimeStamp, 10)) }

		// get the time steps for each position, based on firstUpdateTs
		var timeSteps []int64
		var currentTimeSlot int64 = 0
		var c int64 = 0
		for (c < totalSteps) {
			timeSteps = append(timeSteps, *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c))

			// this will use the next time slot if it is 1ms or more after the start of it
			//if (updateTimeStamp > *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c)) {
			//	currentTimeSlot = c
			//}

			// this will use the next time slot if it is 5s or more after the start of it
			//if (updateTimeStamp > *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c) + 5000) {
			//	currentTimeSlot = c
			//}

			// this will use the next time slot if it is (pct of an interval duration) or more after the start of it
			var pct float64 = .20
			if (updateTimeStamp > *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c) + int64(float64(intervalSeconds * 1000) * pct)) {
				currentTimeSlot = c
			}

			c++
		}

		if debug { fmt.Println("currentTimeSlot: " + strconv.FormatInt(currentTimeSlot, 10)) }

		// now check if this update is in the current time slot or a newer one
		if (updateTimeStamp > timeSteps[rrdPtr.CurrentStep+1]) {
			// this update is in a completely new time slot
			if debug { fmt.Println(ccBlue + "##### NEW STEP ##### this update is in a new step" + ccReset) }

			// set the currentStep to the currentTimeSlot
			rrdPtr.CurrentStep = currentTimeSlot

			// shift the data set
			if (rrdPtr.CurrentStep >= totalSteps-1) {

				// calculate how much to shift by
				var shift int64 = 1
				if (updateTimeStamp >= *rrdPtr.FirstUpdateTs+(totalSteps*intervalSeconds*1000)) {
					// this update needs to shift by more than 1 but obviously not more than the entire data set length
					// because if that were true, the data would have already been reset
					var time_diff int64 = updateTimeStamp - (*rrdPtr.FirstUpdateTs+(totalSteps*intervalSeconds*1000))
					// shift by the number of steps beyond the last considering the original firstUpdateTs
					shift = (time_diff / (intervalSeconds * 1000)) - 1
				}

				if (shift > 0) {

					// shift the data set
					if debug { fmt.Println(ccRed + "shifting data set by: " + strconv.FormatInt(shift, 10) + ccReset) }

					var temp [][]float64
					for e := range rrdPtr.D {

						if (int64(e) >= shift) {
							// append data points more distant than _shift_ to temp
							temp = append(temp, rrdPtr.D[e])
						}

					}

					// copy temp to rrdPtr.D
					copy(rrdPtr.D, temp)

					if (int64(len(rrdPtr.D)) < totalSteps) {
						var p int64 = 0
						// add empty entries to the end
						for (p < totalSteps - int64(len(rrdPtr.D))) {
							rrdPtr.D = append(rrdPtr.D, make([]float64, len(updateDataPoint)))
							p++
						}
					}

					if (dataType == "COUNTER") {
						var temp [][]float64
						for e := range rrdPtr.R {

							if (int64(e) >= shift) {
								// append data points more distant than _shift_ to temp
								temp = append(temp, rrdPtr.R[e])
							}

						}

						// copy temp to rrdPtr.R
						copy(rrdPtr.R, temp)

						if (int64(len(rrdPtr.R)) < totalSteps) {
							var p int64 = 0
							// add empty entries to the end
							for (p < totalSteps - int64(len(rrdPtr.R))) {
								rrdPtr.R = append(rrdPtr.R, make([]float64, len(updateDataPoint)))
								p++
							}
						}
					}

					// add intervalSeconds to firstUpdateTs
					*rrdPtr.FirstUpdateTs = *rrdPtr.FirstUpdateTs+(intervalSeconds*1000*shift)

					rrdPtr.CurrentStep -= shift
					if debug { fmt.Printf("%schanged currentStep: %i%s\n", ccRed, rrdPtr.CurrentStep, ccReset) }

				}
			}

			if (rrdPtr.CurrentStep+1 == totalSteps) {
				// this is needed after a shift of more than 1 but less than totalSteps
				// in case there is an update which is beyond the last when calculated against a new firstUpdateTs that may be milliseconds beyond the previous firstUpdateTs
				rrdPtr.CurrentStep--
			}

			if (rrdPtr.CurrentStep == 0) {
				// there was a buffer calculation that was too greedy and put a new update in the first step
				// which is impossible for a new step, make the current step 1
				rrdPtr.CurrentStep = 1
			}

			if debug { fmt.Println(ccBlue + "inserting data at: " + strconv.FormatInt(rrdPtr.CurrentStep, 10) + ccReset) }

			// remove any data in this step because this is a NEW STEP
			rrdPtr.D[rrdPtr.CurrentStep] = nil
			if (dataType == "COUNTER") {
				rrdPtr.R[rrdPtr.CurrentStep] = nil
			}

			// handle different dataType
			// this is normal processing for an update, assuming there was no previous data missing
			if (dataType == "GAUGE") {

				// insert the data for each data point
				for e := range updateDataPoint {
					rrdPtr.D[rrdPtr.CurrentStep] = append(rrdPtr.D[rrdPtr.CurrentStep], updateDataPoint[e])
				}

				// set the avgCount to 1
				rrdPtr.CurrentAvgCount = 1

			} else if (dataType == "COUNTER") {

				// for each data point
				for e := range updateDataPoint {

					if (rrdPtr.D[rrdPtr.CurrentStep-1] == nil) {

						if debug {
							fmt.Printf("Previous interval is nil\n")
						}

						// only insert the data, there is no previous interval data so no rate can be calculated
						rrdPtr.D[rrdPtr.CurrentStep] = append(rrdPtr.D[rrdPtr.CurrentStep], updateDataPoint[e])

						continue

					}

					// calculate the rate because this is a counter
					// get the value of the interval
					var intervalValue float64 = updateDataPoint[e]-rrdPtr.D[rrdPtr.CurrentStep-1][e]

					// check for a counter reset
					// known by this update value being less than the previous
					if (rrdPtr.D[rrdPtr.CurrentStep-1][e] > updateDataPoint[e]) {

						// the counter has reset, need to check if this happened near the 32 or 64 bit limit
						if debug { fmt.Println(ccBlue + "counter reset" + ccReset) }

						if (rrdPtr.D[rrdPtr.CurrentStep-1][e] < math.MaxUint32 && rrdPtr.D[rrdPtr.CurrentStep-1][e] > math.MaxUint32 * .7) {

							// the last update was between 70% and 100% of the 32 bit uint limit
							// make 32bit adjustments

							// add the remainder of subtracting the last data point from the 32 bit limit to the updateDataPoint
							// use it for rate calculation
							intervalValue = updateDataPoint[e] + math.MaxUint32 - rrdPtr.D[rrdPtr.CurrentStep-1][e]

						} else if (rrdPtr.D[rrdPtr.CurrentStep-1][e] < math.MaxUint64 && rrdPtr.D[rrdPtr.CurrentStep-1][e] > math.MaxUint64 * .7) {

							// the rrd struct number types are currently Float64 (with a limit less than Uint64)
							// this rrd library must be upgraded to use math/big floats anyway

							// the last update was between 70% and 100% of the 64 bit uint limit
							// make 64bit adjustments

							// add the remainder of subtracting the last data point from the 64 bit limit to the updateDataPoint
							// use it for rate calculation
							intervalValue = updateDataPoint[e] + math.MaxUint64 - rrdPtr.D[rrdPtr.CurrentStep-1][e]

						} else {

							// use the update value as intervalValue, showing the rate calculated as the last update being 0
							// because the reset wasn't in a valid range for a counter data type maximum
							intervalValue = updateDataPoint[e]

						}

					}

					if debug { fmt.Println("calculating the rate for " + strconv.FormatFloat(intervalValue, 'f', -1, 64) + " units over " + strconv.FormatInt(intervalSeconds, 10) + " seconds") }
					var rate float64 = intervalValue / float64(intervalSeconds)
					if debug { fmt.Println("inserting data with rate " + strconv.FormatFloat(rate, 'f', -1, 64) + " at time slot " + strconv.FormatInt(rrdPtr.CurrentStep, 10)) }
					rrdPtr.R[rrdPtr.CurrentStep] = append(rrdPtr.R[rrdPtr.CurrentStep], rate)

					// insert the data
					rrdPtr.D[rrdPtr.CurrentStep] = append(rrdPtr.D[rrdPtr.CurrentStep], updateDataPoint[e])

				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType) }
			}

		} else {

			// being here means that this update is in the same step group as the previous
			if debug { fmt.Println("##### SAME STEP ##### this update is in the same step as the previous") }

			// handle different dataType
			if (dataType == "GAUGE") {
				// this update needs to be averaged with the data in this step

				// need to do this for each data point
				for e := range updateDataPoint {

					var avg float64

					// average with a value in the same step
					if debug { fmt.Println("average with a value in the same step") }

					// multiply the avgCount with the existing value
					avg = float64(rrdPtr.CurrentAvgCount) * rrdPtr.D[rrdPtr.CurrentStep][e]

					// add this updateDataPoint
					avg += updateDataPoint[e]

					// increment the avg count
					rrdPtr.CurrentAvgCount++

					// then divide by the avgCount to get the new average
					avg = avg/float64(rrdPtr.CurrentAvgCount)

					if debug { fmt.Println("updating data point with avg " + strconv.FormatFloat(avg, 'f', -1, 64)) }
					rrdPtr.D[rrdPtr.CurrentStep][e] = avg


				}

			} else if (dataType == "COUNTER") {
				// set the counter on this step to that of this update
				for e := range updateDataPoint {
					rrdPtr.D[rrdPtr.CurrentStep][e] = updateDataPoint[e]
				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType) }

			}
		}

		if debug { fmt.Printf("data: %+v\n", rrdPtr.D) }

		if (debug) {
			if (len(rrdPtr.D[rrdPtr.CurrentStep]) != len(updateDataPoint)) {
				// something is wrong
				fmt.Printf("\nDATA LENGTH IS OFF\n\a\a")
			}
		}

	}

}
