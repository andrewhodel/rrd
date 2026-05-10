package main

import (
	"fmt"
	"github.com/andrewhodel/rrd"
)

func main() {

	// Interpolation is a method of constructing (finding) new data points based on the range of a discrete set of known data points.

	// interpolate (Input: 10 megabytes) to the range of an input buffer (BaseRange: 0 bytes to 40 megabytes)
	// to the range of the desired network rate (OutputRange: 1000mbps to 0mbps)
	var interp0 rrd.Interpolation
	interp0.BaseRange = []float64{0, 1000 * 1000 * 40}
	interp0.OutputRange = []float64{1000, 0}
	interp0.Input = 1000 * 1000 * 10

	// SATA 5600 RPM
	//	80 - 150 MB/s

	// SATA 7200 RPM
	//	150 - 250 MB/s

	// SATA SSD
	//	500 - 600 MB/s

	// PCIe 3.0 NVMe
	//	2000 - 3500 MB/s

	// PCIe 4.0 NVMe
	//	3500 - 7000 MB/s

	// interpolate (Input: 70MB/s (sectors written per second + sectors read per second with 512 byte sectors) to the range of a SATA 7200's performance (BaseRange: 0 MB/s to 120 MB/s)
	// to the range of the desired network rate (OutputRange: 1000mbps to 0mbps)
	var interp1 rrd.Interpolation
	interp1.BaseRange = []float64{0, 120}
	interp1.OutputRange = []float64{1000, 0}
	interp1.Input = 70

	// return the output value from the interpolations (interp0, interp1) that is the farthest in the OutputRange (1000 to 0)
	var err, network_interface_rate = rrd.InterpolateValue(true, []rrd.Interpolation{interp0, interp1})

	if (err != nil) {

		fmt.Println("error interpolating network interface.", err.Error())

	} else {

		fmt.Printf("network interface rate from Interpolation: %.3f mbps\n", network_interface_rate)
	}

	if (true) {

		// using a BaseRange that isn't linear

		// this is like mapping to a Curve, you can make BaseRange any type of sequential curve you want, regardless of direction

		// interpolate (Input: 70MB/s (sectors written per second + sectors read per second with 512 byte sectors) to the range of a SATA 7200's performance (BaseRange: 0 MB/s to 120 MB/s)
		// to the range of the desired network rate (OutputRange: 1000mbps to 0mbps)
		var interp rrd.Interpolation
		interp.BaseRange = []float64{0, 100, 120}
		interp.OutputRange = []float64{1000, 0}
		interp.Input = 70

		// return the output value from interp that is the farthest in the OutputRange (1000 to 0)
		var err, network_interface_rate = rrd.InterpolateValue(true, []rrd.Interpolation{interp})

		if (err != nil) {

			fmt.Println("error interpolating network interface.", err.Error())

		} else {

			fmt.Printf("network interface rate from non linear Interpolation: %.3f mbps\n", network_interface_rate)
		}

	}

}
