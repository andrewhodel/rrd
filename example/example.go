package main

import (
	"time"
	"fmt"
	"strconv"
	"github.com/andrewhodel/rrd"
	"io/ioutil"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	var rrdPtr rrd.Rrd

	var update_count = 0

	for (true) {

		fmt.Println("\nupdate number", update_count)

		if (update_count == 5) {
			// simulate missed updates
			fmt.Printf("simulating 12 seconds of missed updates\n")
			time.Sleep(12 * time.Second)
			update_count += 1
			continue
		}

		if (update_count == 15) {
			// perform a recalculate of the rates
			fmt.Printf("recalculating rates")
			rrd.RecalculateRate(8, 10, &rrdPtr)
			update_count += 1
			continue
		}

		dat, err := ioutil.ReadFile("/proc/net/dev")
		check(err)

		//fmt.Println(string(dat))

		s := strings.Split(string(dat), "\n")

		var if_counter = []float64 {0, 0}

		for e := range s {

			n := strings.Fields(s[e])

			if (len(n) < 1) {
				continue
			}

			if (strings.Index(n[0], "enp") != -1 || strings.Index(n[0], "eth") != -1 || strings.Index(n[0], "wlan") != -1 || strings.Index(n[0], "wlp") != -1 || strings.Index(n[0], "lo") != -1) {
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

				fmt.Printf("%s%s: Bytes In: %d\tBytes Out: %d%s\n\n", "\033[34m", n[0], b_in, b_out, "\033[0m")

				break

			}

		}

		rrd.Update(true, 8, 10, "COUNTER", if_counter, &rrdPtr)

		rrd.Dump(&rrdPtr)

		time.Sleep(5 * time.Second)

		update_count += 1

	}

}
