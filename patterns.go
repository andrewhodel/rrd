package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
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

func Equal(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

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

func add_possible_patterns(p []float64, patterns [][]float64, shortest_pattern_len uint64, longest_pattern_len uint64) ([][]float64) {


	var max = longest_pattern_len
	for c := uint64(0); c < max-1; c++ {

		if (max-c < shortest_pattern_len) {
			// only test through the shortest_pattern_len
			break
		}

		// this result is removing the first value each time
		var temp_pattern = p[c:max]
		fmt.Printf("testing %+v\n", temp_pattern)

		if (len(patterns) == 0) {
			// add the first
			patterns = append(patterns, temp_pattern)
		} else {
			patterns = add_if_not_existing(temp_pattern, patterns)
		}

		// a test in the other array direction must happen also to properly find every value
		// only for patterns as long as shortest_pattern_len
		for cc := shortest_pattern_len; cc < uint64(len(temp_pattern)); cc++ {

			var temp_pattern_inner = temp_pattern[0:cc];
			//fmt.Printf("inner testing %+v\n", temp_pattern_inner)
			patterns = add_if_not_existing(temp_pattern_inner, patterns)

		}

	}

	return patterns

}

func get_unique_patterns(d []float64, repeatable_patterns_only bool, shortest_pattern_len uint64) ([][]float64) {
	// return every unique pattern in an array of floats
	//
	// d is an array of floats to find patterns in
	// repeatable_patterns_only set to true is much faster but won't find patterns longer than math.Floor(float64(len(d)/2))

	var max = uint64(len(d));

	if (repeatable_patterns_only) {
		// this is the last entry that would fit inside the data set it came from as a repeating pattern
		// for 10 that would be 5 as it would fit twice
		// for 11 that would be 5 as it would fit twice
		max = uint64(math.Floor(float64(len(d)/2)))
	}

	fmt.Printf("maximum number of patterns per index: %d\n", max);

	// get all the patterns
	var patterns = make([][]float64, 0)
	patterns = add_possible_patterns(d, patterns, shortest_pattern_len, max)

	return patterns

}

func count_patterns_in_set(patterns [][]float64, set []float64) ([]uint64) {

	// return the count of how many times each item in patterns[] exists in set
	var matches = make([]uint64, 0)

	for c := range patterns {

		var inner_matches uint64 = 0

		var pattern = patterns[c]

		for n := 0; n < len(set); n++ {

			if (len(pattern)+n > len(set)) {
				// end of set reached for pattern
				continue
			}

			//fmt.Printf("testing %+v against %+v\n", pattern, set[n:len(set)])

			var match = true;
			for l := range pattern {

				if (pattern[l] != set[l+n]) {
					// not a match
					match = false
					break
				}

			}

			if (match) {
				inner_matches += 1
			}

		}

		//fmt.Printf("testing pattern %+v with %d occurances.\n", pattern, inner_matches)

		matches = append(matches, inner_matches)

	}

	return matches

}

func count_patterns_in_set_fast(patterns [][]float64, set []float64) ([]uint64) {

	//fmt.Printf("\ncounting patterns from:\t\t%+v\n", patterns)
	//fmt.Printf("in set:\t\t\t\t%+v\n\n", set)

	// return the count of how many times each item in patterns[] exists in set
	var matches = make([]uint64, len(patterns))

	// this is for patterns that are beyond c and have already been tested within another pattern
	var sub_pattern_indexes = make([]uint64, 0)
	var starting_pos_in_patterns = 0

	for c := range patterns {

		var sub = false
		for l := range sub_pattern_indexes {
			if (sub_pattern_indexes[l] == uint64(c)) {
				// this pattern (patterns[c]) is already in sub_pattern_indexes
				sub = true
				break
			}
		}
		if (sub) {
			//fmt.Printf("pattern [%d]: %+v is already in sub_pattern_indexes\n", c, patterns[c])
			continue
		}

		var pattern = patterns[c]
		var shortest_pattern_len = len(pattern)

		//fmt.Printf("adding root pattern: %+v\n", pattern)

		// find other patterns that can be tested in this pattern
		// the premise being that
		// 0 1 3 exists in
		// 0 1 3 3
		// 0 1 3 3 7
		// and testing for both rather than repeating a test through everything for each will be faster
		var patterns_within = make([][]float64, 0)
		for n := c+1; n < len(patterns); n++ {

			if (n == c) {
				// this pattern is being tested
				continue
			}

			var is_already = false
			for nn := range sub_pattern_indexes {
				if (sub_pattern_indexes[nn] == uint64(n)) {
					// this pattern has already been tested
					is_already = true
					break
				}
			}

			if (is_already) {
				// this pattern is already tested
				continue
			}

			var shortest = patterns[n]
			var longest = patterns[c]

			if (len(shortest) > len(longest)) {
				longest = patterns[n]
				shortest = patterns[c]
			}

			if (len(shortest) < shortest_pattern_len) {
				// this is used when values are compared
				shortest_pattern_len = len(shortest);
			}

			var is_within = true
			for p := range shortest {
				if (shortest[p] != longest[p]) {
					is_within = false
					break
				}
			}

			if (is_within) {
				sub_pattern_indexes = append(sub_pattern_indexes, uint64(n))
				patterns_within = append(patterns_within, patterns[n])
			}

		}

		// now add pattern to patterns_within
		patterns_within = append(patterns_within, pattern)

		//fmt.Printf("\ntesting %d patterns: %+v\n", len(patterns_within), patterns_within)
		//fmt.Printf("starting position in patterns: %d\n", starting_pos_in_patterns)

		// the first pattern in patterns_within is always the shortest in length
		// and has the same values as each next pattern in patterns_within
		// that continues through each patterns_within entry as such (this is what makes this counting function faster)
		// 0 1
		// 0 1 4
		// 0 1 4 2

		// test each pattern in patterns_within against a shift of the set, named shifting_set
		// this is to look for the pattern at all indexes in the set
		// the shifts look like
		// 0 1 0 0 1 0 1 0 1 0
		// 1 0 0 1 0 1 0 1 0
		// 0 0 1 0 1 0 1 0
		// and so on until
		// 1 0 (because a pattern should be at least 2 values)
		for n := range set {

			var shifting_set = set[n:len(set)]

			if (len(shifting_set) == shortest_pattern_len-1) {
				// a set shift must have at least shortest_pattern_len values
				// this lets the count function work with any length patterns
				// instead of trying all the way to 2
				break
			}

			//fmt.Printf("set shift (%d): %+v\n", len(shifting_set), shifting_set)

			// each pattern in patterns_within grows in length through patterns_within
			// (1) that means we should test for each pattern in patterns_within until the shifting_set length is the length of the pattern in patterns_within
			for nn := range patterns_within {

				//fmt.Printf("\t (%d %d) testing [%d]: %+v against %+v\n", len(patterns_within[nn]), len(shifting_set), nn, patterns_within[nn], shifting_set)

				if (len(patterns_within[nn]) == len(shifting_set)) {
					// (1)
					break
				}

			}

		}

		break

		starting_pos_in_patterns += len(patterns_within)

	}

	return matches

}

func main() {

	// use a long set of values
	var long_count = 400
	var d = make([]float64, long_count)
	for n := 0; n < long_count; n++ {
		var r = rand.Float64()
		d[n] = r
	}

	// shorter testing sets
	//d = []float64 {0,1,0,0,1,0,1,0,1,0}
	d = []float64 {1,2,3,4,5,6,7,8,9,10}

	start0 := time.Now()
	//var patterns = get_unique_patterns(d, true, 4)
	var patterns = get_unique_patterns(d, false, 4)
	duration0 := time.Since(start0)

	start1 := time.Now()
	var counts_fast = count_patterns_in_set_fast(patterns, d)
	duration1 := time.Since(start1)

	start2 := time.Now()
	var counts = count_patterns_in_set(patterns, d)
	duration2 := time.Since(start2)

	fmt.Printf("%d patterns (%dns):\n", len(patterns), duration0)
	fmt.Printf("\ncounts FAST (%dns)\n", duration1)
	fmt.Printf("counts (%dns)\n", duration2)

	/*
	for n := range patterns {
		fmt.Printf("%+v\n", patterns[n])
	}
	fmt.Printf("counts: %+v\n", counts)
	fmt.Printf("counts_fast: %+v\n", counts_fast)
	*/

	_ = counts
	_ = counts_fast

	// from here you can practically do any pattern matching with other data

}
