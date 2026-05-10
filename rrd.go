/*

Copyright 2026 Andrew Hodel
	andrewhodel@gmail.com
	andrew@xyzbots.com

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
	"errors"
	"slices"
)

const (
	// color codes
	colorCodeRed = "\033[31m"
	colorCodeBlue = "\033[34m"
	colorCodeReset = "\033[0m"
)

type Rrd struct {
	D			[][]float64	`bson:"d" json:"d"`
	R			[][]float64	`bson:"r" json:"r"`
	CurrentAvgCount		int64		`bson:"currentAvgCount" json:"currentAvgCount"`
	// use a pointer for FirstUpdateTs to allow nil values
	FirstUpdateTs		*time.Time	`bson:"firstUpdateTs" json:"firstUpdateTs"`
	LastUpdateDataPoint	[]float64	`bson:"lastUpdateDataPoint" json:"lastUpdateDataPoint"`
	MinimumDataPoints	uint64		`bson:"minimumDataPoints" json:"minimumDataPoints"`
}

type Interpolation struct {
	BaseRange		[]float64
	OutputRange		[]float64
	Input			float64
}

func InterpolateValue(interpolations []Interpolation) (error, float64) {

	// return the output value of the interpolation that is the farthest in the OutputRange
	var output_value float64

	for l := range interpolations {

		var interp = interpolations[l]

		// each BaseRange and OutputRange must have at least 2 values
		if (len(interp.BaseRange) < 2 || len(interp.OutputRange) < 2) {
			return errors.New("BaseRange and OutputRange must each have at least 2 values, Interpolation " + strconv.Itoa(l) + " does not."), 0
		}

		// the OutputRange of each Interpolation must be the same
		var outputs_equal = slices.EqualFunc(interpolations[0].OutputRange, interp.OutputRange, func(a, b float64) (bool) {
			return a == b
		})

		if (outputs_equal == false) {
			return errors.New("OutputRange must be the same from each Interpolation."), 0
		}

	}

	return nil, output_value

}

func Avg(rrdPtr *Rrd, index int) (float64) {

	// return the average of all the values in the Rrd at index

	var avg float64
	var count uint64

	if ((*rrdPtr).R != nil) {

		// this is a GAUGE rrd

		for e := range (*rrdPtr).R {
			for n := range (*rrdPtr).R[e] {
				avg += (*rrdPtr).R[e][n]
				count += 1
			}
		}

	} else {

		// this is a COUNTER rrd

		for e := range (*rrdPtr).D {
			for n := range (*rrdPtr).D[e] {
				avg += (*rrdPtr).D[e][n]
				count += 1
			}
		}

	}

	avg = avg / float64(count)

	return avg

}

func Dump(rrdPtr *Rrd) {

	fmt.Printf("rrdPtr: CurrentAvgCount %d, FirstUpdateTs %d\n", (*rrdPtr).CurrentAvgCount, *(*rrdPtr).FirstUpdateTs)
	fmt.Println("rrdPtr LastUpdateDataPoint:")

	for e := range (*rrdPtr).LastUpdateDataPoint {
		fmt.Printf("\t%f", (*rrdPtr).LastUpdateDataPoint[e])
	}

	fmt.Println("")

	fmt.Printf("rrdPtr D (COUNTER VALUES) (%d):\n", len((*rrdPtr).D))

	for e := range (*rrdPtr).D {
		for n := range (*rrdPtr).D[e] {
			fmt.Printf("\tInterval %d\t%f", e, (*rrdPtr).D[e][n])
		}
		fmt.Println("")
	}

	fmt.Println("")

	if ((*rrdPtr).R != nil) {
		fmt.Printf("rrdPtr R (RATE PER SECOND OF COUNTER INTERVALS) (%d):\n", len((*rrdPtr).R))

		for e := range (*rrdPtr).R {
			for n := range (*rrdPtr).R[e] {
				fmt.Printf("\tInterval %d\t%f", e, (*rrdPtr).R[e][n])
			}
			fmt.Println("")
		}

		fmt.Println("")
	}

}

func RecalculateRate(interval time.Duration, totalSteps int64, rrdPtr *Rrd) {

	// recalculate the rate values if the R array exists

	if ((*rrdPtr).R != nil) {

		// for each data point
		for e := range (*rrdPtr).R {

			// reset the rate values
			(*rrdPtr).R[e] = nil

			if (e == 0) {
				// skip the first point set, there is nothing to calculate the rate against
				continue
			}

			if ((*rrdPtr).D[e-1] == nil) {

				// the previous point set has no data
				// go to the next
				continue

			}

			for l := range (*rrdPtr).D[e] {

				var previousPoint = (*rrdPtr).D[e-1][l]
				var currentPoint = (*rrdPtr).D[e][l]

				// get the value of the interval
				var intervalValue float64 = currentPoint - previousPoint

				// check for a counter reset
				// known by this update value being less than the previous
				if (previousPoint > currentPoint) {

					// the counter has reset, need to check if this happened near the 32 or 64 bit limit

					if (previousPoint < math.MaxUint32 && previousPoint > math.MaxUint32 * .7) {

						// the last update was between 70% and 100% of the 32 bit uint limit
						// make 32bit adjustments

						// add the remainder of subtracting the last data point from the 32 bit limit to the currentPoint
						// use it for rate calculation
						intervalValue = currentPoint + math.MaxUint32 - previousPoint

					} else if (previousPoint < math.MaxUint64 && previousPoint > math.MaxUint64 * .7) {

						// the rrd struct number types are currently Float64 (with a limit less than Uint64)
						// this rrd library must be upgraded to use math/big floats anyway

						// the last update was between 70% and 100% of the 64 bit uint limit
						// make 64bit adjustments

						// add the remainder of subtracting the last data point from the 64 bit limit to the currentPoint
						// use it for rate calculation
						intervalValue = currentPoint + math.MaxUint64 - previousPoint

					}

				}

				// set the rate per second as a float
				var rate float64 = intervalValue / float64(interval.Seconds())
				(*rrdPtr).R[e] = append((*rrdPtr).R[e], rate)

			}

		}

	}

}

func Update(debug bool, interval time.Duration, totalSteps int64, dataType string, updateDataPoint []float64, rrdPtr *Rrd) {
	// all timing is based on system time at execution of Update()
	// data can be sent from any time zone, even ones you don't know about yet

	if (updateDataPoint == nil) {
		return
	}

	if (len(updateDataPoint) < int((*rrdPtr).MinimumDataPoints)) {
		if debug { fmt.Printf("updateDataPoint must have at least %d values\n", (*rrdPtr).MinimumDataPoints) }
		return
	} else if (len(updateDataPoint) > int((*rrdPtr).MinimumDataPoints)) {
		// increase the minimum length when updateDataPoint is longer
		(*rrdPtr).MinimumDataPoints = uint64(len(updateDataPoint))

		// make all data point arrays at least the length of this update
		for n := range (*rrdPtr).D {

			if (dataType == "COUNTER") {
				// R values

				if (len((*rrdPtr).R[n]) == 0) {
					// skip [] values
				} else if (len((*rrdPtr).R[n]) != int((*rrdPtr).MinimumDataPoints)) {
					// add zeroes
					var cur_len = len((*rrdPtr).R[n])
					var new_value_count = int((*rrdPtr).MinimumDataPoints) - cur_len
					var l = 0
					for (l < new_value_count) {
						// add a zero for each new field
						(*rrdPtr).R[n] = append((*rrdPtr).R[n], float64(0))
						l = l + 1
					}
				}

			}

			// D values
			if (len((*rrdPtr).D[n]) == 0) {
				// skip [] values
			} else if (len((*rrdPtr).D[n]) != int((*rrdPtr).MinimumDataPoints)) {
				// add zeroes
				var cur_len = len((*rrdPtr).D[n])
				var new_value_count = int((*rrdPtr).MinimumDataPoints) - cur_len
				var l = 0
				for (l < new_value_count) {
					// add a zero for each new field
					(*rrdPtr).D[n] = append((*rrdPtr).D[n], float64(0))
					l = l + 1
				}
			}

		}

	}

	if ((*rrdPtr).FirstUpdateTs == nil) {
		if debug { fmt.Println("FirstUpdateTs is nil") }
	}

	var updateTimeStamp = time.Now()

	// interval - ideal time between updates
	// totalSteps - total steps of data
	// dataType - GAUGE or COUNTER
	//  GAUGE - values that stay within the range of defined integer types, like the value of raw materials.
	//  COUNTER - values that count and can exceed the maximum of a defined integer type.
	// updateTimeStamp - unix epoch timestamp of this update
	// updateDataPoint - data object for this update
	// rrdPtr - data from previous updates
	//
	// returns rrd.Rrd with update added

	if debug { fmt.Println("\n" + colorCodeRed + "### NEW " + dataType + " UPDATE ###" + colorCodeReset) }
	if debug { fmt.Println("interval:", interval) }
	if debug { fmt.Println("totalSteps: " + strconv.FormatInt(totalSteps, 10)) }
	if ((*rrdPtr).FirstUpdateTs != nil) {
		if debug { fmt.Println("firstUpdateTs:", (*(*rrdPtr).FirstUpdateTs), time.Now().Sub((*(*rrdPtr).FirstUpdateTs)), "ago") }
	}
	if debug { fmt.Println("updateTimeStamp:", updateTimeStamp) }
	if debug { fmt.Println("updateDataPoint:") }

	for e := range updateDataPoint {
		if debug { fmt.Printf("\t%f", updateDataPoint[e]) }
	}
	if debug { fmt.Println("") }

	// store updateDataPoint array as lastUpdateDataPoint
	(*rrdPtr).LastUpdateDataPoint = updateDataPoint

	// first need to see if this is the first update or not
	if ((*rrdPtr).FirstUpdateTs == nil) {
		// this is the first update
		if debug { fmt.Println(colorCodeBlue + "### INSERTING FIRST UPDATE ###" + colorCodeReset) }

		// create the array of data points
		(*rrdPtr).D = make([][]float64, totalSteps)
		if (dataType == "COUNTER") {
			(*rrdPtr).R = make([][]float64, totalSteps)
		}

		// insert the data for each data point
		for e := range updateDataPoint {
			if debug { fmt.Printf("\t%f", updateDataPoint[e]) }
			(*rrdPtr).D[0] = append((*rrdPtr).D[0], updateDataPoint[e])
		}
		if debug { fmt.Println("") }

		// set the firstUpdateTs by first allocating space, then assigning the value
		(*rrdPtr).FirstUpdateTs = &updateTimeStamp

	} else {

		// if the updateTimeStamp is later than firstUpdateTs+(totalSteps*interval)
		// or .D has a length of 0
		// it is a new chart
		if (updateTimeStamp.Compare((*(*rrdPtr).FirstUpdateTs).Add(time.Duration(totalSteps * 2) * interval)) >= 0 || len((*rrdPtr).D) == 0) {
			// set firstUpdateTs to nil, this will be considered the first update
			if debug { fmt.Println(colorCodeBlue + "### THIS UPDATE IS NEW ENOUGH TO REPLACE ALL THE DATA ###" + colorCodeReset) }
			(*rrdPtr).FirstUpdateTs = &updateTimeStamp

			// reset all the data
			if (dataType == "COUNTER") {
				// counter types need a rate calculation
				(*rrdPtr).R = nil
				(*rrdPtr).R = make([][]float64, totalSteps)
			}

			(*rrdPtr).D = nil
			(*rrdPtr).D = make([][]float64, totalSteps)

		}

		// this is not the first update
		if debug { fmt.Println(colorCodeBlue + "### PROCESSING " + dataType + " UPDATE ###" + colorCodeReset) }

		// this timestamp
		if debug { fmt.Println("updateTimeStamp:", updateTimeStamp) }

		// get the time steps for each position, based on firstUpdateTs
		var timeSteps []time.Time
		var currentStep int64 = 0
		var c int64 = 0
		for (c < totalSteps) {
			timeSteps = append(timeSteps, (*(*rrdPtr).FirstUpdateTs).Add(interval * time.Duration(c)))

			if (updateTimeStamp.Compare((*(*rrdPtr).FirstUpdateTs).Add(interval * time.Duration(c))) >= 0) {
				currentStep = c
			}

			c++
		}

		// currentTimeSlot will always be now or the newest because the loop iterates totalSteps times
		if debug { fmt.Println("currentStep: " + strconv.FormatInt(currentStep, 10)) }

		// now check if this update is in the current time slot or a newer one
		if (updateTimeStamp.Compare(timeSteps[currentStep]) >= 0 && currentStep != 0) {
			// this update is in a new time slot
			// and it is not the first time slot (multiple updates can happen in the first time slot)
			if debug { fmt.Println(colorCodeBlue + "##### NEW STEP ##### this update is in a new step" + colorCodeReset) }

			// shift the data set
			if (currentStep == totalSteps - 1) {
				// shift the data set

				// calculate how much to shift by
				var shift int64 = 1
				if (updateTimeStamp.Compare((*(*rrdPtr).FirstUpdateTs).Add(time.Duration(totalSteps) * interval)) >= 0) {
					// this update needs to shift by more than 1 time slot
					var time_diff = updateTimeStamp.Sub((*(*rrdPtr).FirstUpdateTs).Add(time.Duration(totalSteps) * interval))

					if debug { fmt.Println("time_diff", time_diff) }

					// shift by the number of steps beyond the last
					shift = (time_diff.Nanoseconds() / interval.Nanoseconds()) - 1
				}

				if debug { fmt.Println(colorCodeRed + "shifting data set by: " + strconv.FormatInt(shift, 10) + colorCodeReset) }

				if (shift > 0) {

					// shift the data set

					var temp = make([][]float64, totalSteps)
					for e := range (*rrdPtr).D {

						if (int64(e) >= shift) {
							// add data points before shift, at their original position - shift
							temp[e - int(shift)] = (*rrdPtr).D[e]
						}

					}

					// copy temp to (*rrdPtr).D
					copy((*rrdPtr).D, temp)

					temp = nil

					if (dataType == "COUNTER") {

						// shift the existing rates

						var temp = make([][]float64, totalSteps)
						for e := range (*rrdPtr).R {

							if (int64(e) >= shift) {
								// add data points before shift, at their original position - shift
								temp[e - int(shift)] = (*rrdPtr).R[e]
							}

						}

						// copy temp to (*rrdPtr).R
						copy((*rrdPtr).R, temp)

						temp = nil

					}

					// set FirstUpdateTs based on shift
					*(*rrdPtr).FirstUpdateTs = (*(*rrdPtr).FirstUpdateTs).Add(interval * time.Duration(shift))

				}

			}

			if debug { fmt.Println(colorCodeBlue + "inserting data at: " + strconv.FormatInt(currentStep, 10) + colorCodeReset) }

			// remove any data in this step because this is a NEW STEP
			(*rrdPtr).D[currentStep] = nil
			if (dataType == "COUNTER") {
				(*rrdPtr).R[currentStep] = nil
			}

			// handle different dataType
			// this is normal processing for an update, assuming there was no previous data missing
			if (dataType == "GAUGE") {

				// insert the data for each data point
				for e := range updateDataPoint {
					(*rrdPtr).D[currentStep] = append((*rrdPtr).D[currentStep], updateDataPoint[e])
				}

				// set the avgCount to 1
				(*rrdPtr).CurrentAvgCount = 1

			} else if (dataType == "COUNTER") {

				// for each data point
				for e := range updateDataPoint {

					if ((*rrdPtr).D[currentStep-1] == nil) {

						if debug {
							fmt.Printf("Previous interval is nil\n")
						}

						// only insert the data, there is no previous interval data to calculate a rate with
						(*rrdPtr).D[currentStep] = append((*rrdPtr).D[currentStep], updateDataPoint[e])

						continue

					}

					// calculate the rate because this is a counter
					// get the value of the interval
					var intervalValue float64 = updateDataPoint[e]-(*rrdPtr).D[currentStep-1][e]

					// check for a counter reset
					// known by this update value being less than the previous
					if ((*rrdPtr).D[currentStep-1][e] > updateDataPoint[e]) {

						// the counter has reset, need to check if this happened near the 32 or 64 bit limit
						if debug { fmt.Println(colorCodeBlue + "counter reset" + colorCodeReset) }

						if ((*rrdPtr).D[currentStep-1][e] < math.MaxUint32 && (*rrdPtr).D[currentStep-1][e] > math.MaxUint32 * .7) {

							// the last update was between 70% and 100% of the 32 bit uint limit
							// make 32bit adjustments

							// add the remainder of subtracting the last data point from the 32 bit limit to the updateDataPoint
							// use it for rate calculation
							intervalValue = updateDataPoint[e] + math.MaxUint32 - (*rrdPtr).D[currentStep-1][e]

						} else if ((*rrdPtr).D[currentStep-1][e] < math.MaxUint64 && (*rrdPtr).D[currentStep-1][e] > math.MaxUint64 * .7) {

							// the rrd struct number types are currently Float64 (with a limit less than Uint64)
							// this rrd library must be upgraded to use math/big floats anyway

							// the last update was between 70% and 100% of the 64 bit uint limit
							// make 64bit adjustments

							// add the remainder of subtracting the last data point from the 64 bit limit to the updateDataPoint
							// use it for rate calculation
							intervalValue = updateDataPoint[e] + math.MaxUint64 - (*rrdPtr).D[currentStep-1][e]

						}

					}

					if debug { fmt.Println("calculating the rate for " + strconv.FormatFloat(intervalValue, 'f', -1, 64) + " units within", interval) }
					// set the rate per second as a float
					var rate float64 = intervalValue / float64(interval.Seconds())
					if debug { fmt.Println("inserting data with rate " + strconv.FormatFloat(rate, 'f', -1, 64) + " per second at time slot " + strconv.FormatInt(currentStep, 10)) }
					(*rrdPtr).R[currentStep] = append((*rrdPtr).R[currentStep], rate)

					// insert the data
					(*rrdPtr).D[currentStep] = append((*rrdPtr).D[currentStep], updateDataPoint[e])

				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType) }
			}

		} else if (len((*rrdPtr).D[currentStep]) == len(updateDataPoint)) {

			// this update is in the same step group as the previous
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
					avg = float64((*rrdPtr).CurrentAvgCount) * (*rrdPtr).D[currentStep][e]

					// add this updateDataPoint
					avg += updateDataPoint[e]

					// increment the avg count
					(*rrdPtr).CurrentAvgCount++

					// then divide by the avgCount to get the new average
					avg = avg/float64((*rrdPtr).CurrentAvgCount)

					if debug { fmt.Println("updating data point with avg " + strconv.FormatFloat(avg, 'f', -1, 64)) }
					(*rrdPtr).D[currentStep][e] = avg


				}

			} else if (dataType == "COUNTER") {
				// set the counter on this step to that of this update
				for e := range updateDataPoint {
					(*rrdPtr).D[currentStep][e] = updateDataPoint[e]
				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType) }

			}
		}

		timeSteps = nil

		if debug { fmt.Printf("data: %+v\n", (*rrdPtr).D) }

		if (debug) {
			if (len((*rrdPtr).D[currentStep]) != len(updateDataPoint)) {
				// something is wrong
				fmt.Printf("\nDATA LENGTH IS OFF\n\a\a")
			}
		}

	}

}
