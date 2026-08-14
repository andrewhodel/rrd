/*

Copyright 2026 Andrew Hodel
	andrew@xyzbots.com
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
	"errors"
	"slices"
	"cmp"
	"sync"
	"os"
	"context"
	"strings"
)

const (

	// color codes
	colorCodeRed = "\033[31m"
	colorCodeBlue = "\033[34m"
	colorCodeReset = "\033[0m"

	// rrd types
	Counter uint8 = 0
	Gauge uint8 = 1

	// token queue types
	Working uint8 = 0
	Instant uint8 = 1

)

type Rrd struct {
	D			[][]*float64	`xyzdb:"D" bson:"D" json:"D"`
	R			[][]*float64	`xyzdb:"R" bson:"R" json:"R"`
	CurrentAvgCount		int64		`xyzdb:"CurrentAvgCount" bson:"CurrentAvgCount" json:"CurrentAvgCount"`
	// use a pointer for FirstUpdateTs to allow nil values
	FirstUpdateTs		*time.Time	`xyzdb:"FirstUpdateTs" bson:"FirstUpdateTs" json:"FirstUpdateTs"`
	LastUpdateDataPoint	[]*float64	`xyzdb:"LastUpdateDataPoint" bson:"LastUpdateDataPoint" json:"LastUpdateDataPoint"`
	LastUpdate		time.Time	`xyzdb:"LastUpdate" bson:"LastUpdate" json:"LastUpdate"`
	MinimumDataPoints	uint64		`xyzdb:"MinimumDataPoints" bson:"MinimumDataPoints" json:"MinimumDataPoints"`
	Interval		time.Duration	`xyzdb:"Interval" bson:"Interval" json:"Interval"`
}

type Interpolation struct {
	// at least 2 values that are sequential, regardless of direction
	// each value is expected to be of an interval with an equal duration
	//	BaseRange 0,100			Input 50			=OutputRatio .5
	//	BaseRange 0,20,100		Input 50			=OutputRatio .6875
	BaseRange		[]float64
	// exactly 2 values
	OutputRange		[]float64
	Input			float64
}

func InterpolateValue(validate_input bool, interpolations []Interpolation) (error, float64) {

	// return the output value of the interpolation that is the farthest in the OutputRange
	var output_value float64
	var first_output_value_set = false

	for l := range interpolations {

		var interp = interpolations[l]

		if (validate_input == true) {

			// each BaseRange must have at least 2 values
			if (len(interp.BaseRange) < 2) {
				return errors.New("Each BaseRange must have at least 2 values, Interpolation " + strconv.Itoa(l) + " does not."), 0
			}

			// each OutputRange must have exactly 2 values
			if (len(interp.OutputRange) != 2) {
				return errors.New("Each OutputRange must have exactly 2 values, Interpolation " + strconv.Itoa(l) + " does not."), 0
			}

			// the OutputRange of each Interpolation must be the same
			var outputs_equal = slices.EqualFunc(interpolations[0].OutputRange, interp.OutputRange, func(a, b float64) (bool) {
				return a == b
			})

			if (outputs_equal == false) {
				return errors.New("OutputRange must be the same from each Interpolation."), 0
			}

		}

		var this_output_value float64 = interp.BaseRange[0]

		var base_range_direction = "asc"
		if (interp.BaseRange[len(interp.BaseRange) - 1] < interp.BaseRange[0]) {
			// BaseRange decreases by farther
			base_range_direction = "desc"
		}

		var output_range_direction = "asc"
		if (interp.OutputRange[1] < interp.OutputRange[0]) {
			// OutputRange decreases by farther
			output_range_direction = "desc"
		}

		if (validate_input == true) {

			if (base_range_direction == "asc") {

				if (slices.IsSorted(interp.BaseRange) == false) {
					return errors.New("BaseRange is not in ascending order."), 0
				}

			} else {

				is_desc := slices.IsSortedFunc(interp.BaseRange, func(a, b float64) (int) {
					return cmp.Compare(b, a)
				})

				if (is_desc == false) {
					return errors.New("BaseRange is not in descending order."), 0
				}

			}

		}

		// a number between 0 and 1 representing how far Input is within OutputRange
		var input_ratio_of_output_range float64

		if (base_range_direction == "asc") {

			if (interp.Input <= interp.BaseRange[0]) {

				// if Input is less than or equal to the first value of BaseRange, Output is the first value of OutputRange
				input_ratio_of_output_range = 0

			} else if (interp.Input >= interp.BaseRange[len(interp.BaseRange) - 1]) {

				// if Input exceeds or is equal to the last value of BaseRange, Output is the last value of OutputRange
				input_ratio_of_output_range = 1

			} else {

				// each value of BaseRange is expected to be of an interval with an equal duration
				var ratio_per_step float64 = 1 / float64(len(interp.BaseRange) - 1)

				for ll := range interp.BaseRange {

					if (ll == 0) {
						continue
					}

					var step_start = interp.BaseRange[ll - 1]
					var step_end = interp.BaseRange[ll]

					if (interp.Input >= step_start && interp.Input < step_end) {

						// Input is within this step

						var step = step_end - step_start

						// a number between 0 and 1 representing how far Input is within this step of BaseRange
						var input_ratio_of_base_range_step float64 = (interp.Input - step_start) / step

						// add input_ratio_of_base_range_step of ratio_per_step to input_ratio_of_output_range
						input_ratio_of_output_range += ratio_per_step * input_ratio_of_base_range_step

						break

					} else {

						// Input is not within this step
						// add ratio of step to input_ratio_of_output_range
						input_ratio_of_output_range += ratio_per_step

					}

				}

			}

		} else {

			if (interp.Input >= interp.BaseRange[0]) {

				// if Input is greater than or equal to the first value of BaseRange, Output is the first value of OutputRange
				input_ratio_of_output_range = 0

			} else if (interp.Input <= interp.BaseRange[len(interp.BaseRange) - 1]) {

				// if Input is less than or is equal to the last value of BaseRange, Output is the last value of OutputRange
				input_ratio_of_output_range = 1

			} else {

				// each value of BaseRange is expected to be of an interval with an equal duration
				var ratio_per_step float64 = 1 / float64(len(interp.BaseRange) - 1)

				slices.Reverse(interp.BaseRange)

				for ll := range interp.BaseRange {

					if (ll == 0) {
						continue
					}

					var step_start = interp.BaseRange[ll - 1]
					var step_end = interp.BaseRange[ll]

					if (interp.Input >= step_start && interp.Input < step_end) {

						// Input is within this step

						var step = step_end - step_start

						// a number between 0 and 1 representing how far Input is within this step of BaseRange
						var input_ratio_of_base_range_step float64 = (interp.Input - step_start) / step

						// add input_ratio_of_base_range_step of ratio_per_step to input_ratio_of_output_range
						input_ratio_of_output_range += ratio_per_step * input_ratio_of_base_range_step

						break

					} else {

						// Input is not within this step
						// add ratio of step to input_ratio_of_output_range
						input_ratio_of_output_range += ratio_per_step

					}

				}

			}

		}

		if (output_range_direction == "asc") {

			// OutputRange[1] is greater than OutputRange[0]
			var output_range_total = interp.OutputRange[1] - interp.OutputRange[0]

			this_output_value = interp.OutputRange[0] + (output_range_total * input_ratio_of_output_range)

			if (first_output_value_set == false) {

				output_value = this_output_value
				first_output_value_set = true

			} else if (this_output_value > output_value) {

				// this_output_value is farther than output_value
				// use it as output_value
				output_value = this_output_value

			}

		} else {

			// OutputRange[0] is greater than OutputRange[1]
			var output_range_total = interp.OutputRange[0] - interp.OutputRange[1]

			this_output_value = interp.OutputRange[0] - (output_range_total * input_ratio_of_output_range)

			if (first_output_value_set == false) {

				output_value = this_output_value
				first_output_value_set = true

			} else if (this_output_value < output_value) {

				// this_output_value is farther than output_value
				// use it as output_value
				output_value = this_output_value

			}

		}

		//fmt.Println("Interpolation", l, "input_ratio_of_output_range", input_ratio_of_output_range, "this_output_value", this_output_value)

	}

	return nil, output_value

}

func Avg(index int, rrdPtrs ...*Rrd) (float64) {

	// return the average of all the values in the Rrd at index

	// if there are multiple Rrd, return the highest value
	// this is explained best in TokenQueue using multiple RRD to track changes

	var highest_avg float64

	for l := range rrdPtrs {

		var rrdPtr = rrdPtrs[l]

		if ((*rrdPtr).D == nil || (*rrdPtr).D[0] == nil) {
			// no data
			// proceed to next rrdPtr
			continue
		}

		var avg float64
		var count float64

		if ((*rrdPtr).R != nil) {

			// this is a Gauge rrd

			for n := range (*rrdPtr).R {

				if (len((*rrdPtr).R[n]) < index + 1) {
					// the rate of this interval has not been calculated yet
					continue
				}

				var r_value_pointer = (*rrdPtr).R[n][index]

				if (r_value_pointer == nil) {
					continue
				}

				avg += (*r_value_pointer)
				count += 1

			}

		} else {

			// this is a Counter rrd

			for n := range (*rrdPtr).D {

				var d_value_pointer = (*rrdPtr).D[n][index]

				if (d_value_pointer == nil) {
					continue
				}

				avg += (*d_value_pointer)
				count += 1
			}

		}

		if (avg > 0) {

			avg = avg / count

		}

		if (l == 0 || avg > highest_avg) {
			highest_avg = avg
		}

	}

	return highest_avg

}

func Dump(rrdPtr *Rrd) {

	if (rrdPtr == nil) {
		fmt.Println("rrdPtr is nil.")
		return
	} else if ((*rrdPtr).FirstUpdateTs == nil) {
		fmt.Println("rrdPtr.FirstUpdateTs is nil, still waiting on first rrd.Update.")
		return
	}

	fmt.Printf("rrdPtr: CurrentAvgCount %d, FirstUpdateTs %d, LastUpdate: %s\n", (*rrdPtr).CurrentAvgCount, (*(*rrdPtr).FirstUpdateTs), (*rrdPtr).LastUpdate.Format(time.Kitchen))
	fmt.Println("rrdPtr LastUpdateDataPoint:")

	for e := range (*rrdPtr).LastUpdateDataPoint {

		if ((*rrdPtr).LastUpdateDataPoint == nil) {
			fmt.Println("\tnil")
		} else {
			fmt.Printf("\t%f\n", (*(*rrdPtr).LastUpdateDataPoint[e]))
		}
	}

	fmt.Printf("rrdPtr D (Gauge or Counter VALUES) (%d):\n", len((*rrdPtr).D))

	for e := range (*rrdPtr).D {

		if ((*rrdPtr).D[e] == nil) {
			fmt.Println("\tInterval", e, "nil")
			continue
		}

		var v string

		for n := range (*rrdPtr).D[e] {

			if ((*rrdPtr).D[e][n] != nil) {
				v += strconv.FormatFloat((*(*rrdPtr).D[e][n]), 'f', 2, 64) + ", "
			} else {
				v += "nil, "
			}

		}

		v = strings.TrimSuffix(v, ", ")

		fmt.Printf("\tInterval %d\t%s\n", e, v)

	}

	if ((*rrdPtr).R != nil) {

		fmt.Printf("rrdPtr R (RATE PER SECOND OF Counter INTERVALS) (%d):\n", len((*rrdPtr).R))

		for e := range (*rrdPtr).R {

			if ((*rrdPtr).R[e] == nil) {
				fmt.Println("\tInterval", e, "nil")
				continue
			}

			var v string

			for n := range (*rrdPtr).R[e] {

				if ((*rrdPtr).R[e][n] != nil) {
					v += strconv.FormatFloat((*(*rrdPtr).R[e][n]), 'f', 2, 64) + ", "
				} else {
					v += "nil, "
				}

			}

			v = strings.TrimSuffix(v, ", ")

			fmt.Printf("\tInterval %d\t%s\n", e, v)

		}

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

			var steps_between int = 1
			var skip = false

			for {

				if (e - steps_between == -1) {
					// no previous interval has data
					skip = true
					break
				} else if ((*rrdPtr).D[e-steps_between] == nil) {

					// the previous interval (point set) has no data
					// go to the next
					continue

				}

			}

			if (skip == true) {
				// no previous interval has data
				continue
			}

			for l := range (*rrdPtr).D[e - steps_between] {

				var previous_value_pointer = (*rrdPtr).D[e - steps_between][l]
				var current_value_pointer = (*rrdPtr).D[e][l]

				if (previous_value_pointer == nil || current_value_pointer == nil) {
					continue
				}

				// get the value of the interval
				var intervalValue float64 = (*current_value_pointer) - (*previous_value_pointer)

				// check for a counter reset
				// known by this update value being less than the previous
				if ((*previous_value_pointer) > (*current_value_pointer)) {

					// the counter has reset, need to check if this happened near the 32 or 64 bit limit

					if ((*previous_value_pointer) < math.MaxUint32 && (*previous_value_pointer) > math.MaxUint32 * .7) {

						// the last update was between 70% and 100% of the 32 bit uint limit
						// make 32bit adjustments

						// add the remainder of subtracting the last data point from the 32 bit limit to the (*current_value_pointer)
						// use it for rate calculation
						intervalValue = (*current_value_pointer) + math.MaxUint32 - (*previous_value_pointer)

					} else if ((*previous_value_pointer) < math.MaxUint64 && (*previous_value_pointer) > math.MaxUint64 * .7) {

						// the rrd struct number types are currently Float64 (with a limit less than Uint64)
						// this rrd library must be upgraded to use math/big floats anyway

						// the last update was between 70% and 100% of the 64 bit uint limit
						// make 64bit adjustments

						// add the remainder of subtracting the last data point from the 64 bit limit to the (*current_value_pointer)
						// use it for rate calculation
						intervalValue = (*current_value_pointer) + math.MaxUint64 - (*previous_value_pointer)

					}

				}

				// set the rate per second as a float
				var rate float64 = intervalValue / (float64(interval.Seconds()) * float64(steps_between))

				for {

					var interval = e + 1 - steps_between

					if (len((*rrdPtr).R[interval]) <= e) {
						// add rate
						(*rrdPtr).R[interval] = append((*rrdPtr).R[interval], &rate)
					} else {
						// change existing rate
						(*rrdPtr).R[interval][e] = &rate
					}

					if (steps_between == 1) {
						// all intervals updated
						break
					}

					steps_between -= 1

				}

			}

		}

	}

}

func Update(debug bool, interval time.Duration, totalSteps int64, dataType uint8, updateDataPoint []*float64, rrdPtr *Rrd) {

	// all timing is based on system time at execution of Update
	// data can be sent from any time zone, even ones you don't know about yet

	var updateTimeStamp = time.Now()

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

			if (dataType == Counter) {

				// Counter

				// R values

				if (len((*rrdPtr).R[n]) == 0) {
					// skip [] values
				} else if (len((*rrdPtr).R[n]) != int((*rrdPtr).MinimumDataPoints)) {
					// add nil values
					var cur_len = len((*rrdPtr).R[n])
					var new_value_count = int((*rrdPtr).MinimumDataPoints) - cur_len
					var l = 0
					for (l < new_value_count) {
						// add nil for each new field
						(*rrdPtr).R[n] = append((*rrdPtr).R[n], nil)
						l = l + 1
					}
				}

			}

			// D values
			if (len((*rrdPtr).D[n]) == 0) {
				// skip [] values
			} else if (len((*rrdPtr).D[n]) != int((*rrdPtr).MinimumDataPoints)) {
				// add nil values
				var cur_len = len((*rrdPtr).D[n])
				var new_value_count = int((*rrdPtr).MinimumDataPoints) - cur_len
				var l = 0
				for (l < new_value_count) {
					// add nil for each new field
					(*rrdPtr).D[n] = append((*rrdPtr).D[n], nil)
					l = l + 1
				}
			}

		}

	}

	if ((*rrdPtr).FirstUpdateTs == nil) {
		if debug { fmt.Println("FirstUpdateTs is nil") }
	}

	// interval - ideal time between updates
	// totalSteps - total steps of data
	// dataType - rrd.Gauge or rrd.Counter
	//  Gauge - values that stay within the range of defined integer types, like the value of raw materials.
	//  Counter - values that count and can exceed the maximum of a defined integer type.
	// updateTimeStamp - unix epoch timestamp of this update
	// updateDataPoint - data object for this update
	// rrdPtr - data from previous updates
	//
	// returns rrd.Rrd with update added

	if debug {

		fmt.Println("\n" + colorCodeRed + "### NEW " + dataType_string(dataType) + " UPDATE ###" + colorCodeReset)
		fmt.Println("interval:", interval)
		fmt.Println("totalSteps: " + strconv.FormatInt(totalSteps, 10))
		if ((*rrdPtr).FirstUpdateTs != nil) {
			fmt.Println("firstUpdateTs:", (*(*rrdPtr).FirstUpdateTs), updateTimeStamp.Sub((*(*rrdPtr).FirstUpdateTs)), "ago")
		}
		fmt.Println("updateTimeStamp:", updateTimeStamp)
		fmt.Println("updateDataPoint:")

		for e := range updateDataPoint {

			if (updateDataPoint[e] == nil) {
				fmt.Println("\tnil")
			} else {
				fmt.Printf("\t%f\n", (*updateDataPoint[e]))
			}

		}

	}

	// store updateDataPoint array as lastUpdateDataPoint
	(*rrdPtr).LastUpdateDataPoint = updateDataPoint
	(*rrdPtr).LastUpdate = updateTimeStamp

	// first need to see if this is the first update or not
	if ((*rrdPtr).FirstUpdateTs == nil) {

		// this is the first update
		if debug { fmt.Println(colorCodeBlue + "### INSERTING FIRST UPDATE ###" + colorCodeReset) }

		// create the array of data points
		(*rrdPtr).D = make([][]*float64, totalSteps)
		if (dataType == Counter) {
			// Counter
			(*rrdPtr).R = make([][]*float64, totalSteps)
		}

		// insert the data for each data point
		for e := range updateDataPoint {

			if debug {
				if (updateDataPoint[e] == nil) {
					fmt.Println("\tnil")
				} else {
					fmt.Printf("\t%f\n", (*updateDataPoint[e]))
				}
			}

			(*rrdPtr).D[0] = append((*rrdPtr).D[0], updateDataPoint[e])

		}

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
			if (dataType == Counter) {
				// counter types need a rate calculation
				(*rrdPtr).R = nil
				(*rrdPtr).R = make([][]*float64, totalSteps)
			}

			(*rrdPtr).D = nil
			(*rrdPtr).D = make([][]*float64, totalSteps)

		}

		// this is not the first update
		if debug { fmt.Println(colorCodeBlue + "### PROCESSING " + dataType_string(dataType) + " UPDATE ###" + colorCodeReset) }

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

					var temp = make([][]*float64, totalSteps)
					for e := range (*rrdPtr).D {

						if (int64(e) >= shift) {
							// add data points before shift, at their original position - shift
							temp[e - int(shift)] = (*rrdPtr).D[e]
						}

					}

					// copy temp to (*rrdPtr).D
					copy((*rrdPtr).D, temp)

					temp = nil

					if (dataType == Counter) {

						// Counter

						// shift the existing rates

						var temp = make([][]*float64, totalSteps)
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
			if (dataType == Counter) {
				// Counter
				(*rrdPtr).R[currentStep] = nil
			}

			// handle different dataType
			// this is normal processing for an update, assuming there was no previous data missing
			if (dataType == Gauge) {

				// Gauge

				// insert the data for each data point
				for e := range updateDataPoint {
					(*rrdPtr).D[currentStep] = append((*rrdPtr).D[currentStep], updateDataPoint[e])
				}

				// set the avgCount to 1
				(*rrdPtr).CurrentAvgCount = 1

			} else if (dataType == Counter) {

				// Counter

				// for each data point
				for e := range updateDataPoint {

					// add the updateDataPoint to D
					(*rrdPtr).D[currentStep] = append((*rrdPtr).D[currentStep], updateDataPoint[e])

					var steps_between int64 = 1

					if ((*rrdPtr).D[currentStep - steps_between] == nil) {

						// the previous interval has no data
						// find the one before it that does

						var skip = false

						for {

							if (currentStep - steps_between == -1) {
								// no previous interval has data
								skip = true
								break
							} else if ((*rrdPtr).D[currentStep - steps_between] == nil) {
								// this previous interval does not have data
								steps_between += 1
								continue
							}

							// this interval has data
							break

						}

						if debug {
							fmt.Printf("Previous %d intervals are nil.\n", steps_between)
						}

						if (skip == true) {
							// no previous interval has data
							continue
						}

					} else if (updateDataPoint[e] == nil) {

						// there is no way to calculate a rate from a nil value
						continue

					}

					// calculate the rate because this is a counter
					// get the value of the interval
					var intervalValue float64 = (*updateDataPoint[e]) - (*(*rrdPtr).D[currentStep - steps_between][e])

					// check for a counter reset
					// known by this update value being less than the previous
					if ((*(*rrdPtr).D[currentStep - steps_between][e]) > (*updateDataPoint[e])) {

						// the counter has reset, need to check if this happened near the 32 or 64 bit limit
						if debug { fmt.Println(colorCodeBlue + "counter reset" + colorCodeReset) }

						if ((*(*rrdPtr).D[currentStep - steps_between][e]) < math.MaxUint32 && (*(*rrdPtr).D[currentStep - steps_between][e]) > math.MaxUint32 * .7) {

							// the last update was between 70% and 100% of the 32 bit uint limit
							// make 32bit adjustments

							// add the remainder of subtracting the last data point from the 32 bit limit to the updateDataPoint
							// use it for rate calculation
							intervalValue = (*updateDataPoint[e]) + math.MaxUint32 - (*(*rrdPtr).D[currentStep - steps_between][e])

						} else if ((*(*rrdPtr).D[currentStep - steps_between][e]) < math.MaxUint64 && (*(*rrdPtr).D[currentStep - steps_between][e]) > math.MaxUint64 * .7) {

							// the rrd struct number types are currently Float64 (with a limit less than Uint64)
							// this rrd library must be upgraded to use math/big floats anyway

							// the last update was between 70% and 100% of the 64 bit uint limit
							// make 64bit adjustments

							// add the remainder of subtracting the last data point from the 64 bit limit to the updateDataPoint
							// use it for rate calculation
							intervalValue = (*updateDataPoint[e]) + math.MaxUint64 - (*(*rrdPtr).D[currentStep - steps_between][e])

						}

					}

					if debug { fmt.Println("calculating the rate for " + strconv.FormatFloat(intervalValue, 'f', -1, 64) + " units within", interval) }

					// set the rate per second as a float
					var rate float64 = intervalValue / (float64(interval.Seconds()) * float64(steps_between))

					for {

						var interval int64 = currentStep + 1 - steps_between

						if debug { fmt.Println("inserting data with rate " + strconv.FormatFloat(rate, 'f', -1, 64) + " per second at time slot " + strconv.FormatInt(interval, 10)) }

						if (len((*rrdPtr).R[interval]) <= e) {
							// add rate
							(*rrdPtr).R[interval] = append((*rrdPtr).R[interval], &rate)
						} else {
							// change existing rate
							(*rrdPtr).R[interval][e] = &rate
						}

						if (steps_between == 1) {
							// all intervals updated
							break
						}

						steps_between -= 1

					}

				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType_string(dataType)) }
			}

		} else if (len((*rrdPtr).D[currentStep]) == len(updateDataPoint)) {

			// this update is in the same step group as the previous
			if debug { fmt.Println("##### SAME STEP ##### this update is in the same step as the previous") }

			// handle different dataType
			if (dataType == Gauge) {

				// Gauge

				// this update needs to be averaged with the data in this step

				// need to do this for each data point
				for e := range updateDataPoint {

					if (updateDataPoint[e] == nil) {
						// a nil value shouldn't remove existing nil values of the same step
						continue
					}

					var avg float64

					// average with a value in the same step
					if debug { fmt.Println("average with a value in the same step") }

					// multiply the avgCount with the existing value
					avg = float64((*rrdPtr).CurrentAvgCount) * (*(*rrdPtr).D[currentStep][e])

					// add this updateDataPoint
					avg += (*updateDataPoint[e])

					// increment the avg count
					(*rrdPtr).CurrentAvgCount++

					// then divide by the avgCount to get the new average
					avg = avg/float64((*rrdPtr).CurrentAvgCount)

					if debug { fmt.Println("updating data point with avg " + strconv.FormatFloat(avg, 'f', -1, 64)) }
					(*rrdPtr).D[currentStep][e] = &avg


				}

			} else if (dataType == Counter) {

				// Counter

				// set the counter on this step to that of this update
				for e := range updateDataPoint {

					if (updateDataPoint[e] == nil) {
						// a nil value shouldn't remove existing nil values of the same step
						continue
					}

					(*rrdPtr).D[currentStep][e] = updateDataPoint[e]
				}

			} else {
				if debug { fmt.Println("unsupported dataType " + dataType_string(dataType)) }

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

func dataType_string(dataType uint8) (string) {

	if (dataType == Counter) {
		return "Counter"
	}

	return "Gauge"

}

type TokenQueue struct {
	// the sum of each Token.Size allowed per second
	RatePerSecond			float64
	Sum				uint64
	Tokens				[]*Token
	// 0 works with QueueToken and UnqueueToken
	// 1 works with WaitToCompleteToken
	Type				uint8
	// the first second
	FSRrdPointer			*Rrd
	RrdPointer			*Rrd
	LongRrdPointer			*Rrd
	Cond				*sync.Cond
	Cancel				context.CancelFunc
	sync.RWMutex
}

type Token struct {
	// the size of this token, write size in bytes for example
	Size				uint64
	Pending				bool
	// the lowest Prio proceeds first
	// this allows allocation and order to be handled conceptually at each Token
	Prio				uint64
	Time				time.Time
	MaxWait				*time.Duration
}

type LockLess struct {
}
func (ll *LockLess) Lock() {
}
func (ll *LockLess) Unlock() {
}

func SetTokenQueueLimiter(token_queue_pointer **TokenQueue, rate_per_second float64, token_queue_type uint8) {

	if (*token_queue_pointer == nil) {

		// create a new TokenQueue

		var token_queue TokenQueue
		token_queue.RatePerSecond = rate_per_second
		token_queue.Type = token_queue_type

		// create RRD
		var first_second_rrd_pointer Rrd
		token_queue.FSRrdPointer = &first_second_rrd_pointer
		var rrd_pointer Rrd
		token_queue.RrdPointer = &rrd_pointer
		var long_rrd_pointer Rrd
		token_queue.LongRrdPointer = &long_rrd_pointer

		// create Cond
		var cond_mutex LockLess
		token_queue.Cond = sync.NewCond(&cond_mutex)

		*token_queue_pointer = &token_queue

	} else {

		// update the existing TokenQueue

		(**token_queue_pointer).Lock()

		(**token_queue_pointer).RatePerSecond = rate_per_second
		(**token_queue_pointer).Type = token_queue_type

		// create RRD
		var first_second_rrd_pointer Rrd
		(**token_queue_pointer).FSRrdPointer = &first_second_rrd_pointer
		var rrd_pointer Rrd
		(**token_queue_pointer).RrdPointer = &rrd_pointer
		var long_rrd_pointer Rrd
		(**token_queue_pointer).LongRrdPointer = &long_rrd_pointer

		(**token_queue_pointer).Unlock()

	}

	// start goroutine to Update RRD

	(**token_queue_pointer).Lock()

	if ((**token_queue_pointer).Cancel == nil) {

		var ctx context.Context
		ctx, (**token_queue_pointer).Cancel = context.WithCancel(context.Background())

		go func() {

			for {

				if (ctx.Err() != nil) {

					// user called TokenQueue.Cancel
					// end the RRD update loop

					// nil Cancel in order that another is created if SetTokenQueueLimiter is ran again
					(**token_queue_pointer).Cancel = nil

					return

				}

				(**token_queue_pointer).Lock()

				// update the RRD to keep track of the rate
				Update(false, time.Millisecond * 10, 100, Counter, GetUpdateValues(float64((*token_queue_pointer).Sum)), (*token_queue_pointer).FSRrdPointer)
				Update(false, time.Second, 5, Counter, GetUpdateValues(float64((*token_queue_pointer).Sum)), (*token_queue_pointer).RrdPointer)
				Update(false, time.Second, 60, Counter, GetUpdateValues(float64((*token_queue_pointer).Sum)), (*token_queue_pointer).LongRrdPointer)

				(**token_queue_pointer).Unlock()

				(*(**token_queue_pointer).Cond).Broadcast()

				time.Sleep(time.Millisecond * 170)

			}

		}()

	}

	(**token_queue_pointer).Unlock()

}

func GetUpdateValues(values ...any) ([]*float64) {

	// used to create a []*float64 for rrd.Update that does not access original data but accepts nil values

	var retval []*float64

	for _, value := range values {

		switch v := value.(type) {
			case float64:
				var vv = v
				retval = append(retval, &vv)
			case *float64:
				var vv = (*v)
				retval = append(retval, &vv)
			case nil:
				retval = append(retval, nil)
			default:
				fmt.Printf("rrd.GetUpdateValues requires parameters that are float64, *float64 or nil.  value is of type: %T\n", v)
				os.Exit(1)
		}

	}

	return retval

}

func WaitToken(token_queue_pointer *TokenQueue, size uint64, prio uint64, max_wait *time.Duration) {

	// called to make the token wait based on the rate and number of other WaitToken calls of the same TokenQueue
	// when it's known that writing the data is instant
	// for example, writing to a socket buffer that is immediately sent to another router's buffer

	// only for TokenQueue.Type == rrd.Instant

	var token Token
	token.Size = size
	token.Prio = prio
	token.Time = time.Now()
	token.MaxWait = max_wait

	for {

		// this is the average per 10ms
		(*token_queue_pointer).RLock()
		var first_second_avg = Avg(0, (*token_queue_pointer).FSRrdPointer)
		(*token_queue_pointer).RUnlock()

		if (first_second_avg * 100 > (*token_queue_pointer).RatePerSecond) {
			time.Sleep(time.Millisecond * 170)
			continue
		}

		break

	}

	(*token_queue_pointer).Lock()

	if ((*token_queue_pointer).Type != Instant) {

		fmt.Println("WaitToken requires TokenQueue.Type == rrd.Instant.")
		os.Exit(1)

	} else if ((*token_queue_pointer).RatePerSecond == 0) {

		// there is no rate set, do not block
		(*token_queue_pointer).Unlock()
		return

	}

	(*token_queue_pointer).Tokens = append((*token_queue_pointer).Tokens, &token)

	(*token_queue_pointer).Unlock()

	for {

		if (token.MaxWait != nil && time.Now().Sub(token.Time) >= (*token.MaxWait)) {

			// this token has a MaxWait that is at least now
			break

		}

		(*token_queue_pointer).RLock()

		if (Avg(0, (*token_queue_pointer).LongRrdPointer, (*token_queue_pointer).RrdPointer) > (*token_queue_pointer).RatePerSecond) {

			// the rate is higher than TokenQueue.RatePerSecond
			// wait

		} else {

			// the token can proceed according the RRD rate

			// find the lowest Prio
			var lowest_prio uint64 = math.MaxUint64
			var prios = make(map[uint64] []*Token, 10)

			for l := range (*token_queue_pointer).Tokens {

				var this_token_pointer = (*token_queue_pointer).Tokens[l]

				if ((*this_token_pointer).Prio <= lowest_prio) {

					// this token has equal to or lower than the lowest Prio

					prios[(*this_token_pointer).Prio] = append(prios[(*this_token_pointer).Prio], this_token_pointer)
					lowest_prio = (*this_token_pointer).Prio

				}

			}

			(*token_queue_pointer).RUnlock()

			if (token.Prio == lowest_prio && &token == prios[token.Prio][0]) {

				// this token has the lowest Prio
				// and is the first in sequence of this Prio

				(*token_queue_pointer).Lock()

				(*token_queue_pointer).Sum += token.Size

				// there are 100 10ms intervals in 1 second
				Update(false, time.Millisecond * 10, 100, Counter, GetUpdateValues(float64((*token_queue_pointer).Sum)), (*token_queue_pointer).FSRrdPointer)

				for l := range (*token_queue_pointer).Tokens {

					var this_token_pointer = (*token_queue_pointer).Tokens[l]

					if (this_token_pointer == &token) {

						// delete the token
						(*token_queue_pointer).Tokens = slices.Delete((*token_queue_pointer).Tokens, l, l + 1)
						break

					}

				}

				(*token_queue_pointer).Unlock()

				return

			}

		}

		(*token_queue_pointer).RUnlock()

		(*(*token_queue_pointer).Cond).Wait()

	}

	(*(*token_queue_pointer).Cond).Broadcast()

}

func QueueToken(token_queue_pointer *TokenQueue, size uint64, prio uint64, max_wait *time.Duration) (*Token) {

	// called to place a token in the token queue
	// and blocks until the token can begin

	// only for TokenQueue.Type == rrd.Working

	var token Token
	token.Size = size
	token.Prio = prio
	token.Time = time.Now()
	token.MaxWait = max_wait

	var only_token = false

	(*token_queue_pointer).Lock()

	if ((*token_queue_pointer).Type != Working) {

		fmt.Println("QueueToken requires TokenQueue.Type == rrd.Working.")
		os.Exit(1)

	} else if ((*token_queue_pointer).RatePerSecond == 0) {

		// there is no rate set, do not block or return a pointer
		(*token_queue_pointer).Unlock()
		return nil

	}

	(*token_queue_pointer).Tokens = append((*token_queue_pointer).Tokens, &token)

	if (len((*token_queue_pointer).Tokens) == 1) {

		// the token added is the only token
		only_token = true

	}

	(*token_queue_pointer).Unlock()

	for {

		// this is the average per 10ms
		(*token_queue_pointer).RLock()
		var first_second_avg = Avg(0, (*token_queue_pointer).FSRrdPointer)
		(*token_queue_pointer).RUnlock()

		if (first_second_avg * 100 > (*token_queue_pointer).RatePerSecond) {
			time.Sleep(time.Millisecond * 170)
			continue
		}

		break

	}

	if (only_token == false || Avg(0, (*token_queue_pointer).LongRrdPointer, (*token_queue_pointer).RrdPointer) > (*token_queue_pointer).RatePerSecond) {

		for {

			if (token.MaxWait != nil && time.Now().Sub(token.Time) >= (*token.MaxWait)) {

				// this token has a MaxWait that is at least now
				break

			}

			// find the rate of the pending tokens
			var pending_tokens_rate uint64

			// find the Time of the first token that is Pending
			var first_pending_time *time.Time

			// find the lowest Prio
			var lowest_prio uint64 = math.MaxUint64
			var prios = make(map[uint64] []*Token, 10)

			(*token_queue_pointer).RLock()

			var rrd_avg = Avg(0, (*token_queue_pointer).LongRrdPointer, (*token_queue_pointer).RrdPointer)

			for l := range (*token_queue_pointer).Tokens {

				var this_token_pointer = (*token_queue_pointer).Tokens[l]

				if ((*this_token_pointer).Pending == true) {

					// this token is Pending

					if (first_pending_time == nil) {
						first_pending_time = &(*this_token_pointer).Time
					}

					pending_tokens_rate += (*this_token_pointer).Size

				} else if ((*this_token_pointer).Prio <= lowest_prio) {

					// this token is not Pending and has equal to or lower than the lowest Prio

					prios[(*this_token_pointer).Prio] = append(prios[(*this_token_pointer).Prio], this_token_pointer)
					lowest_prio = (*this_token_pointer).Prio

				}

			}

			if (pending_tokens_rate == 0 && rrd_avg < (*token_queue_pointer).RatePerSecond) {

				// no tokens are pending, start without wait
				(*token_queue_pointer).RUnlock()

				break

			}

			(*token_queue_pointer).RUnlock()

			if (token.Prio == lowest_prio && &token == prios[token.Prio][0]) {

				// this token has the lowest Prio
				// and is the first in sequence of this Prio

				// the token created by this QueueToken is the best token to let be Pending

				if (first_pending_time == nil) {

					// there are no Pending tokens

					if (rrd_avg < (*token_queue_pointer).RatePerSecond) {

						// the rate per second of the TokenQueue has not been breached overall
						break

					}

				} else {

					// get the pending_tokens_rate
					var duration_since_first_pending_token = time.Now().Sub((*first_pending_time))
					var actual_pending_tokens_rate = float64(pending_tokens_rate) / duration_since_first_pending_token.Seconds()

					if ((*token_queue_pointer).RatePerSecond > rrd_avg + actual_pending_tokens_rate) {

						// the rate per second of the TokenQueue has not been breached by Pending tokens
						// the rate per second of the TokenQueue has not been breached overall
						break

					}

				}

			}

			// 200 microsecond iterations and 80MB per iteration is 400GB/s
			// and 1 million concurrent QueueToken calls would use all 5ghz of one CPU core running at 5ghz
			(*(*token_queue_pointer).Cond).Wait()

		}

	}

	(*token_queue_pointer).Lock()

	(*token_queue_pointer).Sum += token.Size

	token.Pending = true

	// there are 100 10ms intervals in 1 second
	Update(false, time.Millisecond * 10, 100, Counter, GetUpdateValues(float64((*token_queue_pointer).Sum)), (*token_queue_pointer).FSRrdPointer)

	(*token_queue_pointer).Unlock()

	return &token

}

func UnqueueToken(token_queue_pointer *TokenQueue, token_pointer *Token) {

	// called once the token has been used by a slow process that relies on the token queue rate
	// mostly writing to disk or many gigabytes being written to RAM

	(*token_queue_pointer).Lock()

	for l := range (*token_queue_pointer).Tokens {

		var this_token_pointer = (*token_queue_pointer).Tokens[l]

		if (this_token_pointer == token_pointer) {

			// delete the token
			(*token_queue_pointer).Tokens = slices.Delete((*token_queue_pointer).Tokens, l, l + 1)
			break

		}

	}

	(*token_queue_pointer).Unlock()

	(*(*token_queue_pointer).Cond).Broadcast()

}
