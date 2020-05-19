/*

Copyright 2020 Andrew Hodel
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

func Dump(rrdPtr *Rrd) {

	fmt.Printf("rrdPtr: CurrentStep %d, CurrentAvgCount %d, FirstUpdateTs %d\n", rrdPtr.CurrentStep, rrdPtr.CurrentAvgCount, *rrdPtr.FirstUpdateTs)
	fmt.Println("rrdPtr LastUpdateDataPoint:")

	for e := range rrdPtr.LastUpdateDataPoint {
		fmt.Printf("\t%f", rrdPtr.LastUpdateDataPoint[e])
	}

	fmt.Println("")

	fmt.Printf("rrdPtr D (%d):\n", len(rrdPtr.D))

	for e := range rrdPtr.D {
		for n := range rrdPtr.D[e] {
			fmt.Printf("\t%f", rrdPtr.D[e][n])
		}
		fmt.Println("")
	}

	fmt.Println("")

	if (rrdPtr.R != nil) {
		fmt.Printf("rrdPtr R (%d):\n", len(rrdPtr.R))

		for e := range rrdPtr.R {
			for n := range rrdPtr.R[e] {
				fmt.Printf("\t%f", rrdPtr.R[e][n])
			}
			fmt.Println("")
		}

		fmt.Println("")
	}

}

func Update(intervalSeconds int64, totalSteps int64, dataType string, updateDataPoint []float64, rrdPtr *Rrd) {

	var debug = false

	if (updateDataPoint == nil) {
		return
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
	//  COUNTER - things that count up, if we get a value that's less than last time it means it reset... stored as a per second rate
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

	if debug { Dump(rrdPtr) }

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
			// set firstUpdateTs to nil so this will be considered the first update
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

	// first we need to see if this is the first update or not
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
			if debug { fmt.Println(updateDataPoint[e]) }
			rrdPtr.D[0] = append(rrdPtr.D[0], updateDataPoint[e])
		}

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

			if (updateTimeStamp >= *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c)) {
				currentTimeSlot = c
			}

			c++
		}

		if debug { fmt.Println("currentTimeSlot: " + strconv.FormatInt(currentTimeSlot, 10)) }

		// now check if this update is in the current time slot
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
					if debug { fmt.Println(ccRed + "changed currentStep: " + string(rrdPtr.CurrentStep) + ccReset) }

				}
			}

			if (rrdPtr.CurrentStep+1 == totalSteps) {
				// this is needed after a shift of more than 1 but less than totalSteps
				// in case there is an update which is beyond the last when calculated against a new firstUpdateTs that may be milliseconds beyond the previous firstUpdateTs
				rrdPtr.CurrentStep--
			}

			if debug { fmt.Println(ccBlue + "inserting data at: " + strconv.FormatInt(rrdPtr.CurrentStep, 10) + ccReset) }

			// handle different dataType
			if (dataType == "GAUGE") {

				// insert the data for each data point
				for e := range updateDataPoint {
					rrdPtr.D[rrdPtr.CurrentStep][e] = updateDataPoint[e]
				}

				// set the avgCount to 1
				rrdPtr.CurrentAvgCount = 1

			} else if (dataType == "COUNTER") {

				// for each data point
				for e := range updateDataPoint {

					// we need to check for overflow, overflow happens when a counter resets so we check the last values to see if they were close to the limit if the previous update
					// is 3 times the size or larger, meaning if the current update is 33% or smaller it's probably an overflow
					if (rrdPtr.D[rrdPtr.CurrentStep-1][e] > updateDataPoint[e]*3) {

						// oh no, the counter has overflowed so we need to check if this happened near 32 or 64 bit limit
						if debug { fmt.Println(ccBlue + "overflow" + ccReset) }

						// the 32 bit limit is 2,147,483,647 so we should check if we were within 10% of that either way on the last update
						if (rrdPtr.D[rrdPtr.CurrentStep][e]<(2147483647*.1)-2147483647) {
							// this was so close to the limit that we are going to make 32bit adjustments
							// for this calculation we just need to add the remainder of subtracting the last data point from the 32 bit limit to the updateDataPoint
							updateDataPoint[e] += 2147483647-rrdPtr.D[rrdPtr.CurrentStep-1][e]

							// the 64 bit limit is 9,223,372,036,854,775,807 so we should check if we were within 1% of that
						} else if (rrdPtr.D[rrdPtr.CurrentStep][e]<(9223372036854775807*.01)-9223372036854775807) {
							// this was so close to the limit that we are going to make 64bit adjustments
							// for this calculation we just need to add the remainder of subtracting the last data point from the 64 bit limit to the updateDataPoint
							updateDataPoint[e] += 9223372036854775807-rrdPtr.D[rrdPtr.CurrentStep-1][e]

						}
					}

					// for a counter, we need to divide the difference of this step and the previous step by
					// the difference in seconds between the updates
					var rate float64 = updateDataPoint[e]-rrdPtr.D[rrdPtr.CurrentStep-1][e]
					if debug { fmt.Println("calculating the rate for " + strconv.FormatFloat(rate, 'f', -1, 64) + " units over " + strconv.FormatInt(intervalSeconds, 10) + " seconds") }
					rate = rate / float64(intervalSeconds)
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
				// this update needs to be averaged with the last

				// we need to do this for each data point
				for e := range updateDataPoint {

					var avg float64

					if (rrdPtr.CurrentAvgCount > 1) {
						// we are averaging with a previous update that was itself an average
						if debug { fmt.Println("we are averaging with a previous update that was itself an average") }

						// that means we have to multiply the avgCount of the previous update by the data point of the previous update
						if (rrdPtr.CurrentStep == 0) {
							// this is the first update, we need to average with currentStep not the previous step
							avg = float64(rrdPtr.CurrentAvgCount) * rrdPtr.D[rrdPtr.CurrentStep][e]
						} else {
							avg = float64(rrdPtr.CurrentAvgCount) * rrdPtr.D[rrdPtr.CurrentStep-1][e]
						}
						// add this updateDataPoint
						avg += updateDataPoint[e]
						// increment the avg count
						rrdPtr.CurrentAvgCount++
						// then divide by the avgCount
						avg = avg/float64(rrdPtr.CurrentAvgCount)

						if debug { fmt.Println("updating data point with avg " + strconv.FormatFloat(avg, 'f', -1, 64)) }
						rrdPtr.D[rrdPtr.CurrentStep][e] = avg

					} else {
						// we need to average the previous update with this one
						if debug { fmt.Println("averaging with previous update") }

						// we need to add the previous update data point to this one then divide by 2 for the average
						avg = (updateDataPoint[e]+rrdPtr.D[rrdPtr.CurrentStep][e])/2
						// set the avgCount to 2
						rrdPtr.CurrentAvgCount = 2
						// and insert it
						if debug { fmt.Println("updating data point with avg " + strconv.FormatFloat(avg, 'f', -1, 64)) }
						rrdPtr.D[rrdPtr.CurrentStep][e] = avg

					}

				}

			} else if (dataType == "COUNTER") {
				// increase the counter on the last update to this one for each data point
				// this actually means to modify, not increase because it would be an increased value
				for e := range updateDataPoint {
					rrdPtr.D[rrdPtr.CurrentStep][e] = updateDataPoint[e]
				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType) }

			}
		}

	}

}
