package main

import (
	"fmt"
	"math"
)

/*

4 points

all patterns are
0,1 (last repeatable pattern length for this data)
0,1,2
0,1,2,3

1,2
1,2,3

2,3

*/

func add_if_not_existing(p []float64, patterns [][]float64) ([][]float64) {

	var exists = false
	for c := range patterns {

		if (len(patterns[c]) != len(p)) {
			// we can be sure this is not a match
			// and not waste more cycles
			continue
		} else {
			// test for a match
			var matching_set = true
			for cc := 0; cc < len(p); cc++ {
				if (patterns[c][cc] != p[cc]) {
					// not a matching value
					// meaning the set cannot match
					matching_set = false
					break
				}
			}

			if (matching_set) {
				exists = true
				break
			}
		}

	}

	if (!exists) {
		// add it
		patterns = append(patterns, p)
	}

	return patterns

}

func add_possible_patterns(p []float64, patterns [][]float64) ([][]float64) {

	var max int = len(p)
	for c := 0; c < max-1; c++ {

		// result is removing the first value each time
		var temp_pattern = p[c:max]
		//fmt.Printf("testing %+v\n", temp_pattern)

		if (len(patterns) == 0) {
			// add the first
			//fmt.Printf("adding the first max_len_pattern to patterns\n")
			patterns = append(patterns, temp_pattern)
		} else {
			patterns = add_if_not_existing(temp_pattern, patterns)
		}

		// a test in the other array direction must happen also to properly find every value
		for cc := 2; cc < len(temp_pattern)-1; cc++ {

			var temp_pattern_inner = temp_pattern[0:cc];
			//fmt.Printf("inner testing %+v\n", temp_pattern_inner)
			patterns = add_if_not_existing(temp_pattern_inner, patterns)

		}

	}

	return patterns

}

func get_unique_patterns(d []float64, repeatable_patterns_only bool) ([][]float64) {
	// return every unique pattern in an array of floats
	//
	// d is an array of floats to find patterns in
	// repeatable_patterns_only set to true is much faster but won't find patterns longer than math.Floor(float64(len(d)/2))

	var max int = len(d);

	if (repeatable_patterns_only) {
		// this is the last entry that would fit inside the data set it came from as a repeating pattern
		// for 10 that would be 5 as it would fit twice
		// for 11 that would be 5 as it would fit twice
		max = int(math.Floor(float64(len(d)/2)))
	}

	//fmt.Printf("maximum number of patterns per index: %d\n", max);

	var max_len_patterns = make([][]float64, 0)

	for e := 0; e < len(d)-1; e++ {

		var inner_max = max
		if (len(d)-e <= max) {
			inner_max = len(d)-e
		}
		inner_max -= 1

		//fmt.Printf("from %d to %d\n", e, e+inner_max)

		// starting at each index of d get a data set from e to e+inner_max
		// that is the maximum length pattern for each index of the original data set
		var max_len_pattern = make([]float64, 0)
		for c := e; c<=e+inner_max; c++ {

			max_len_pattern = append(max_len_pattern, d[c])
			//fmt.Printf("%d\n", c)

		}

		if (len(max_len_patterns) == 0) {
			// add the first max_len_pattern
			max_len_patterns = append(max_len_patterns, max_len_pattern)
		}

		// make sure this max_len_pattern does not already exist in another max_len_pattern
		// there is no purpose in using it to find inner patterns if it is already in a longer max_len_pattern
		var new_matches_existing = false
		for c := range max_len_patterns {

			var same = true

			// the newest max_len_pattern is always shortest, test it against each other max_len_pattern
			for cc := range max_len_pattern {
				if (max_len_pattern[cc] != max_len_patterns[c][cc]) {
					same = false
					break
				}
			}

			if (same) {
				new_matches_existing = true
				break
			}

		}

		if (!new_matches_existing) {
			// this max_len_pattern is unique, it is safe to generate sub patterns from it
			max_len_patterns = append(max_len_patterns, max_len_pattern)
		}

	}

	// get all the patterns from each max_len_patterns entry
	var patterns = make([][]float64, 0)
	for e := range max_len_patterns {

		//fmt.Printf("\nmax_len_pattern: %+v\n", max_len_patterns[e])

		patterns = add_possible_patterns(max_len_patterns[e], patterns)

	}

	return patterns

}

func main() {

	//var d = []float64 {0,1,0,0,1,0,1,0,1,0}
	var d = []float64 {1,2,3,4,5,6,7,8,9,10}

	//var patterns = get_unique_patterns(d, true)
	var patterns = get_unique_patterns(d, false)
	fmt.Printf("patterns: %+v\n", patterns)

	// from here you can practically do any pattern matching with other data

}
