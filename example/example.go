package main

import (
	"time"
	"fmt"
	"strconv"
	"github.com/andrewhodel/rrd"
	"io/ioutil"
	"strings"
	"slices"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	var if_rrd rrd.Rrd
	if_rrd.Interval = time.Second
	if_rrd.TotalSteps = 10
	if_rrd.DataType = rrd.Counter
	if_rrd.Debug = true

	var update_count = 0

	for {

		fmt.Println("\nupdate number", update_count)

		if (update_count == 65) {

			// miss 3 seconds of updates
			fmt.Printf("\x1b[1;31m3 seconds of missed updates...\x1b[0m\n")
			time.Sleep(3 * time.Second)
			update_count += 1
			continue

		} else if (update_count == 15) {

			fmt.Println("recalculating rates")
			rrd.RecalculateRate(&if_rrd)
			update_count += 1
			continue

		}

		dat, err := ioutil.ReadFile("/proc/net/dev")
		check(err)

		//fmt.Println(string(dat))

		s := strings.Split(string(dat), "\n")

		slices.SortFunc(s, func(a, b string) (int) {

			n := strings.Fields(a)

			if (len(n) < 1) {
				return 1
			}

			if (strings.Index(n[0], "enp") != -1 || strings.Index(n[0], "eth") != -1 || strings.Index(n[0], "wlan") != -1) {
				return -1
			}

			return 1
		})

		// must be []any to work with rrd.GetUpdateValues
		var if_counter = []any{float64(0), float64(0)}
		var if_name string

		for e := range s {

			n := strings.Fields(s[e])

			if (len(n) < 1) {
				continue
			}

			if (strings.Index(n[0], "en") != -1 || strings.Index(n[0], "eth") != -1 || strings.Index(n[0], "wlan") != -1 || strings.Index(n[0], "wlp") != -1 || strings.Index(n[0], "lo") != -1) {

				// bytes in
				b_in, err := strconv.Atoi(n[1])
				check(err)
				// bytes out
				b_out, err := strconv.Atoi(n[9])
				check(err)

				if (b_in == 0 && b_out == 0) {
					continue
				}

				if_counter[0] = float64(b_in)
				if_counter[1] = float64(b_out)

				if_name = strings.TrimSuffix(n[0], ":")

				if (strings.Index(n[0], "lo") == -1) {

					// look for another interface if localhost
					break

				}

			}

		}

		fmt.Printf("\x1b[34m%s\x1b[0m\n", if_name)

		rrd.Update(rrd.GetUpdateValues(if_counter...), &if_rrd)

		rrd.Dump(&if_rrd)

		time.Sleep(time.Millisecond * 500)

		update_count += 1

	}

}
