package main

import (
	"time"
	//"fmt"
	//"strconv"
	"go-rrd/rrd"
)

func main() {

	var jsonDb rrd.Rrd

	for (true) {
		var dp = []float64 {10, 12}

		rrd.Update(8, 10, "COUNTER", dp, &jsonDb)

		time.Sleep(5 * time.Second)
	}

}
