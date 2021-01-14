/*

Copyright 2021 Andrew Hodel
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
	"math/big"
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
	CurrentStep		int64		`bson:"currentStep" json:"currentStep"`
	CurrentAvgCount		int64		`bson:"currentAvgCount" json:"currentAvgCount"`
	// use a pointer for FirstUpdateTs so can check for nil
	FirstUpdateTs		*int64		`bson:"firstUpdateTs" json:"firstUpdateTs"`
	LastUpdateDataPoint	[]float64	`bson:"lastUpdateDataPoint" json:"lastUpdateDataPoint"`
	// Overflow fixes the problem of many overflows sent in a single step as math/big is used to calculate the rate from multiple float64 values
	// as long as an interface isn't faster than 1.7976931348623157e+308 bits per second the rate will never be incorrect
	// D needs to be changed to a uint64 data type to support 64 bit counters which means rrd.Update() needs to accept and store multiple data types
	Overflow		[][]float64	`bson:"overflow" json:"overflow"`
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

	var debug = true

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
			// set firstUpdateTs to nil so this will be considered the first update
			// which resets all the data
			if debug { fmt.Println(ccBlue + "### THIS UPDATE IS SO MUCH NEWER THAN THE EXISTING DATA THAT IT REPLACES IT ###" + ccReset) }
			rrdPtr.FirstUpdateTs = nil

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
			// create the Overflow tracker
			rrdPtr.Overflow = make([][]float64, len(updateDataPoint))
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

		// set the CurrentStep to 0
		rrdPtr.CurrentStep = 0

	} else {

		// this is not the first update
		if debug { fmt.Println(ccBlue + "### PROCESSING " + dataType + " UPDATE ###" + ccReset) }

		// this timestamp
		if debug { fmt.Println("updateTimeStamp: " + strconv.FormatInt(updateTimeStamp, 10)) }

		// this overflow
		if debug { fmt.Printf("overflow: %v\n", rrdPtr.Overflow) }

		// get the time steps for each position, based on firstUpdateTs
		var timeSteps []int64
		var currentTimeSlot int64 = 0
		var c int64 = 0
		for (c < totalSteps) {
			timeSteps = append(timeSteps, *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c))

			// this will use the next time slot if it is only 1ms after the start of it
			//if (updateTimeStamp > *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c)) {
			//	currentTimeSlot = c
			//}

			// this will use the next time slot if it is received more than 5 seconds into it
			//if (updateTimeStamp > *rrdPtr.FirstUpdateTs + (intervalSeconds * 1000 * c) + 5000) {
			//	currentTimeSlot = c
			//}

			// this will use the time slot if it is received (pct) of time through the next one or in this one
			var pct float64 = .20
			// this seems like the most reasonable solution with network latency being a factor
			// it could be 50% or 70% really, but it seems that we should be discarding data that is that behind
			// especially considering that the delay could be processing   -
			//								--
			//								--
			//								---
			//								---
			//								---
			// the explanation is that updates can be sent without knowledge of time on the far side, as if you were receiving data from an unknown source or distant planet
			// it needs to be that way because many devices with a tcp/ip stack don't have a ntp client
			// or NTP isn't accurate enough at a high resolution
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
				// which is impossible for a new step, so make the current step 1
				rrdPtr.CurrentStep = 1
			}

			if debug { fmt.Println(ccBlue + "inserting data at: " + strconv.FormatInt(rrdPtr.CurrentStep, 10) + ccReset) }

			// if the previous {total_missing_to_fill} points are missing data, fill them in with this updates data
			// the problem with this is not that bad, it could increase the update interval by a multiple of {total_missing_to_fill}
			// but it allows situations like a 5 ms update interval and an update every 6ms to continue properly showing data for counters
			// which is a reasonable expectation in network software

			// the time slot calculation already has a buffer (some % after intervalSeconds into the next step) which resolves most of this

			// the reason you want to update the last {total_missing_to_fill} data points rather than just one, is because you could have a situation where 10 data points were sent
			// expecting .1 second intervals yet took 2 seconds for the network to provide them
			// this could result in ugly data and probably should be set based on expected network latency, realized latency peaks and step interval
			// especially when using intervals that are milliseconds or nanoseconds apart

			// you could just take this block out if you knew for a fact that data would arrive on time (processing and collection on the same hardware)
			// of course if you knew that for a fact, you wouldn't have to worry about this block executing

			// I have explained it below (EXPLANATION) for the GAUGE and COUNTER types

			var total_missing_to_fill int64 = 1
			var l int64 = total_missing_to_fill
			for (l > 0) {

				if (rrdPtr.CurrentStep - l < 0) {
					// first entry was already filled in
					break
				}

				if (len(rrdPtr.D[rrdPtr.CurrentStep - l]) != len(updateDataPoint)) {
					// this previous set of data points wasn't filled in
					// use this data point
					for e := range updateDataPoint {
						rrdPtr.D[rrdPtr.CurrentStep - l] = append(rrdPtr.D[rrdPtr.CurrentStep - l], updateDataPoint[e])
					}
					if debug { fmt.Printf("PREVIOUS DATA STEP WAS MISSING\n\a\a") }

				}

				l--
			}

			// remove any data in this step
			// otherwise append will continue adding data to each step
			// there would have been data in a step after a shift
			rrdPtr.D[rrdPtr.CurrentStep] = nil
			if (dataType == "COUNTER") {
				rrdPtr.R[rrdPtr.CurrentStep] = nil
			}

			// handle different dataType
			// this is normal processing for an update, assuming there was no previous data missing
			if (dataType == "GAUGE") {

				// EXPLANATION
				// if data is missing in every other update, charts would always look terrible
				// D D D D D N D N D N D N D

				// best to fill in the values only if <3 were missing, to show real data outages
				// this works because 5 minute updates would require quite a bit of loss to make a 2 day data point go missing

				// insert the data for each data point
				for e := range updateDataPoint {
					rrdPtr.D[rrdPtr.CurrentStep] = append(rrdPtr.D[rrdPtr.CurrentStep], updateDataPoint[e])
				}

				// set the avgCount to 1
				rrdPtr.CurrentAvgCount = 1

			} else if (dataType == "COUNTER") {

				// EXPLANATION
				// if data is missing in every other update, rates would never be calculated
				// D D D D D N D N D N D N D
				// R R R R R N N N N N N N N

				// with a counter, you can always fill in the data

				// fill in the previous point with the data from this update
				// not all, only the last 1 because we want to be able to show update outages

				// for each data point
				for e := range updateDataPoint {

					big_d := new(big.Float).SetFloat64(float64(updateDataPoint[e]))

					// add all the previous overflows to prev
					big_prev := new(big.Float).SetFloat64(float64(0.0))
					for n := range rrdPtr.Overflow[e] {
						big_prev.Add(big_prev, new(big.Float).SetFloat64(rrdPtr.Overflow[e][n]))
					}

					// reset Overflow
					rrdPtr.Overflow[e] = nil

					if (rrdPtr.D[rrdPtr.CurrentStep-1][e] > updateDataPoint[e]) {

						// counter overflow, check if this happened near 32 or 64 bit limit
						if debug { fmt.Println(ccBlue + "new step overflow" + ccReset) }

						// the 32 bit limit for an unsigned integer is 4,294,967,295
						// check if the last step was that or less
						if (rrdPtr.D[rrdPtr.CurrentStep-1][e] <= math.MaxUint32) {
							// make 32bit overflow adjustments
							if debug { fmt.Println(ccBlue + "32 bit overflow" + ccReset) }

							// add the remainder of subtracting the last data point from the limit to the updateDataPoint
							big_d.Add(big_d, new(big.Float).SetFloat64(float64(2147483647) - float64(rrdPtr.D[rrdPtr.CurrentStep-1][e])))

						} else {
							// the 64 bit limit is 18,446,744,073,709,551,615
							// make 64bit overflow adjustments
							if debug { fmt.Println(ccBlue + "64 bit overflow" + ccReset) }

							// add the remainder of subtracting the last data point from the limit to the updateDataPoint
							big_d.Add(big_d, new(big.Float).SetFloat64(float64(math.MaxUint64) - float64(rrdPtr.D[rrdPtr.CurrentStep-1][e])))

						}

						// store the overflow, if big_d > the float64 limit then store multiple float64 values
						if (big_d.Cmp(new(big.Float).SetFloat64(float64(math.MaxUint64))) == 1) {
							// divide big_d by the limit
							v := new(big.Float).Quo(big_d, new(big.Float).SetFloat64(float64(math.MaxUint64)))

							f, accuracy := v.Float64()
							_ = accuracy

							// get the floor value of f
							floor := math.Floor(f)

							// get the remainder
							rem := f - floor

							// store floor values of the limit
							var vc int = 0
							for (vc < int(floor)) {
								rrdPtr.Overflow[e] = append(rrdPtr.Overflow[e], float64(math.MaxUint64))
								vc++
							}

							// store 1 value of the limit * the remainder
							rrdPtr.Overflow[e] = append(rrdPtr.Overflow[e], float64(math.MaxUint64 * rem))

						} else {
							// store the overflow
							v, accuracy := big_d.Float64()
							_ = accuracy
							rrdPtr.Overflow[e] = append(rrdPtr.Overflow[e], float64(v))
						}

					}

					// for a counter, need to divide the difference of this step and the previous step by
					// the difference in seconds between the updates

					// add the last step and big_prev to get the "last step"
					big_prev.Add(big_prev, new(big.Float).SetFloat64(rrdPtr.D[rrdPtr.CurrentStep-1][e]))
					// subtract big_prev from big_d to get the units between
					big_d.Add(big_d, big_prev.Neg(big_prev))

					diff, accuracy := big_d.Float64()
					_ = accuracy

					diff = math.Abs(diff)

					if debug { fmt.Println("calculating the rate for " + strconv.FormatFloat(diff, 'f', -1, 64) + " units over " + strconv.FormatInt(intervalSeconds, 10) + " seconds") }
					var rate float64 = diff / float64(intervalSeconds)
					if debug { fmt.Println("inserting data with rate " + strconv.FormatFloat(rate, 'f', -1, 64) + " at time slot " + strconv.FormatInt(rrdPtr.CurrentStep, 10)) }
					rrdPtr.R[rrdPtr.CurrentStep] = append(rrdPtr.R[rrdPtr.CurrentStep], rate)

					// insert the data
					rrdPtr.D[rrdPtr.CurrentStep] = append(rrdPtr.D[rrdPtr.CurrentStep], updateDataPoint[e])

				}

				var l int64 = total_missing_to_fill
				for (l > 0) {

					if (rrdPtr.CurrentStep - l < 0) {
						// first entry was already filled in
						break
					}

					if (len(rrdPtr.R[rrdPtr.CurrentStep - l]) != len(updateDataPoint)) {
						// this previous set of data points wasn't filled in
						// use the rate that was calculated
						for e := range updateDataPoint {
							rrdPtr.R[rrdPtr.CurrentStep - l] = append(rrdPtr.R[rrdPtr.CurrentStep - l], rrdPtr.R[rrdPtr.CurrentStep][e])
						}
						if debug { fmt.Printf("PREVIOUS RATE DATA STEP WAS MISSING\n\a\a") }

					}

					l--
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

					big_d := new(big.Float).SetFloat64(float64(updateDataPoint[e]))

					// check for counter overflow in the same step

					if (rrdPtr.D[rrdPtr.CurrentStep][e] > updateDataPoint[e]) {

						// counter overflow, check if this happened near 32 or 64 bit limit
						if debug { fmt.Println(ccBlue + "same step overflow" + ccReset) }

						// the 32 bit limit for an unsigned integer is 4,294,967,295
						if (rrdPtr.D[rrdPtr.CurrentStep][e] <= math.MaxUint32) {
							// make 32bit overflow adjustments
							if debug { fmt.Println(ccBlue + "32 bit overflow" + ccReset) }

							// add the remainder of subtracting this data point from the limit to the updateDataPoint
							big_d.Add(big_d, new(big.Float).SetFloat64(float64(2147483647) - float64(rrdPtr.D[rrdPtr.CurrentStep][e])))

						} else {
							// the 64 bit limit is 18,446,744,073,709,551,615
							// make 64bit overflow adjustments
							if debug { fmt.Println(ccBlue + "64 bit overflow" + ccReset) }

							// add the remainder of subtracting this data point from the limit to the updateDataPoint
							big_d.Add(big_d, new(big.Float).SetFloat64(float64(math.MaxUint64) - float64(rrdPtr.D[rrdPtr.CurrentStep][e])))

						}

						// store the overflow, if big_d > the float64 limit then store multiple float64 values
						if (big_d.Cmp(new(big.Float).SetFloat64(float64(math.MaxUint64))) == 1) {
							// divide big_d by the limit
							v := new(big.Float).Quo(big_d, new(big.Float).SetFloat64(float64(math.MaxUint64)))

							f, accuracy := v.Float64()
							_ = accuracy

							// get the floor value of f
							floor := math.Floor(f)

							// get the remainder
							rem := f - floor

							// store floor values of the limit
							var vc int = 0
							for (vc < int(floor)) {
								rrdPtr.Overflow[e] = append(rrdPtr.Overflow[e], float64(math.MaxUint64))
								vc++
							}

							// store 1 value of the limit * the remainder
							rrdPtr.Overflow[e] = append(rrdPtr.Overflow[e], float64(math.MaxUint64 * rem))

						} else {
							// store the overflow
							v, accuracy := big_d.Float64()
							_ = accuracy
							rrdPtr.Overflow[e] = append(rrdPtr.Overflow[e], float64(v))
						}

					}

					// store the data point
					rrdPtr.D[rrdPtr.CurrentStep][e] = updateDataPoint[e]
				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType) }

			}
		}

		if debug { fmt.Printf("data: %+v\n", rrdPtr.D) }

	}

}
