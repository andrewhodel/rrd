package main

import (
	"fmt"
	"github.com/andrewhodel/rrd"
	"sync/atomic"
	"io"
	mrand "math/rand/v2"
	"time"
)

func main() {

	// use a ramdisk if your disk is not fast enough

	// create a rrd.TokenQueue that limits disk writing at 15MB/s
	var token_queue_pointer *rrd.TokenQueue
	rrd.SetTokenQueueLimiter(&token_queue_pointer, 15 * 1000 * 1000, rrd.Working)

	fmt.Println("\n\nStarting 120 goroutines each writing data to a \x1b[1;33mrrd.TokenQueue limited to 15MB/s\x1b[0m each using a \x1b[1;34mrrd.Token.Size that is random between 0 and 200KB\x1b[0m.")

	var token_size atomic.Uint64
	// first token size 4KB
	token_size.Store(4000)

	fmt.Println("\n\x1b[1;31mfirst token size:", rrd.Bytes_to_size_string(4000), "\x1b[0m\n\n")

	// print the rate
	go func() {

		for {

			(*token_queue_pointer).RLock()

			var avg = rrd.Avg(0, (*token_queue_pointer).RrdPointer)
			var mid_avg = rrd.Avg(0, (*token_queue_pointer).MidRrdPointer)
			var long_avg = rrd.Avg(0, (*token_queue_pointer).LongRrdPointer)

			var interval = time.Duration((*(*token_queue_pointer).RrdPointer).Interval * time.Duration((*(*token_queue_pointer).RrdPointer).TotalSteps)).String()
			var mid_interval = time.Duration((*(*token_queue_pointer).MidRrdPointer).Interval * time.Duration((*(*token_queue_pointer).MidRrdPointer).TotalSteps)).String()
			var long_interval = time.Duration((*(*token_queue_pointer).LongRrdPointer).Interval * time.Duration((*(*token_queue_pointer).LongRrdPointer).TotalSteps)).String()

			(*token_queue_pointer).RUnlock()

			fmt.Printf("Token size: %-20slast %s rate: %-20slast %s rate: %-20slast %s rate: %-20s\n", rrd.Bytes_to_size_string(token_size.Load()), interval, rrd.Bytes_to_size_string(uint64(avg)) + "/s", mid_interval, rrd.Bytes_to_size_string(uint64(mid_avg)), long_interval, rrd.Bytes_to_size_string(uint64(long_avg)) + "/s")

			time.Sleep(time.Millisecond * 200)

		}

	}()

	var write_goroutines uint64
	for {

		go func() {

			for {

				var write_size = uint64(mrand.IntN(200000))

				var data_to_write = make([]byte, write_size)

				var token_pointer = rrd.QueueToken(token_queue_pointer, write_size, 0, nil)

				// write to /dev/null (io.Discard works with non Posix OS)
				io.Discard.Write(data_to_write)

				rrd.UnqueueToken(token_queue_pointer, token_pointer)

			}

		}()

		write_goroutines += 1

		if (write_goroutines == 120) {

			// have 120 writing goroutines
			break

		}

	}

	select{}

}
