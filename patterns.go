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
		for cc := 2; cc < len(temp_pattern); cc++ {

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
			continue
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

	fmt.Printf("\ncounting patterns from:\t\t%+v\n", patterns)
	fmt.Printf("in set:\t\t\t\t%+v\n\n", set)

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
			fmt.Printf("pattern [%d]: %+v is already in sub_pattern_indexes", c, patterns[c])
			continue
		}

		var pattern = patterns[c]

		fmt.Printf("adding root pattern: %+v\n", pattern)

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

		fmt.Printf("\ntesting %d patterns: %+v\n", len(patterns_within), patterns_within)
		fmt.Printf("starting position in patterns: %d\n", starting_pos_in_patterns)

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
			fmt.Printf("set shift: %+v\n", shifting_set)

			// each pattern in patterns_within grows in length through patterns_within
			// (1) that means we should test for each pattern in patterns_within until the shifting_set length is 1 less than the pattern of patterns_within length
			for nn := range patterns_within {

				fmt.Printf("\t (%d %d) testing [%d]: %+v\n", len(patterns_within[nn]), len(shifting_set), nn, patterns_within[nn])

				if (len(patterns_within[nn])-1 > len(shifting_set)) {
					// (1)
					//break
				}

			}

		}

		starting_pos_in_patterns += len(patterns_within)

		break

	}

	fmt.Printf("\n")

	return matches

}

func main() {

	//var d = []float64 {0,1,0,0,1,0,1,0,1,0}
	var d = []float64 {1,2,3,4,5,6,7,8,9,10}

	//var patterns = get_unique_patterns(d, true)
	var patterns = get_unique_patterns(d, false)

	fmt.Printf("patterns (%d):\n", len(patterns))
	for n := range patterns {
		fmt.Printf("%+v\n", patterns[n])
	}

	var counts = count_patterns_in_set_fast(patterns, d)
	fmt.Printf("counts (%d): %+v\n", len(counts), counts)

	// from here you can practically do any pattern matching with other data

}
