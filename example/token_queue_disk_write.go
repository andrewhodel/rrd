package main

import (
	"fmt"
	"github.com/andrewhodel/rrd"
	"sync/atomic"
	"io"
	mrand "math/rand/v2"
	"time"
	"sync"
)

func main() {

	// use a ramdisk if your disk is not fast enough

	// create a rrd.TokenQueue that limits disk writing at 15MB/s
	var token_queue_pointer *rrd.TokenQueue
	rrd.SetTokenQueueLimiter(&token_queue_pointer, 15 * 1000 * 1000, rrd.Working)

	fmt.Println("Starting 3 goroutines each writing data to a rrd.TokenQueue using a rrd.Token.Size that changes every 30 seconds and is limited at 15MB/s.")

	var token_size atomic.Uint64
	// first token size 4KB
	token_size.Store(4000)
	fmt.Println("\x1b[1;31mfirst token size:", Bytes_to_size_string(4000), "\x1b[0m\n\n")

	// TokenQueue stores the rate of the last 5s internally, keep the rate of the last minute also
	var lastm_rrd rrd.Rrd
	var lastm_mutex sync.RWMutex
	var lastm_counter uint64

	// print the rate
	go func() {

		for {

			lastm_mutex.RLock()
			var mavg = rrd.Avg(&lastm_rrd, 0)
			lastm_mutex.RUnlock()

			(*token_queue_pointer).RLock()
			var avg = rrd.Avg((*token_queue_pointer).RrdPointer, 0)
			(*token_queue_pointer).RUnlock()

			fmt.Printf("Token size: %-20slast 5s rate: %-20slast 1m rate: %-20s\n", Bytes_to_size_string(token_size.Load()), Bytes_to_size_string(uint64(avg)) + "/s", Bytes_to_size_string(uint64(mavg)) + "/s")

			time.Sleep(time.Millisecond * 200)

		}

	}()

	// every 30 seconds use a different token size
	go func() {

		for {

			time.Sleep(time.Second * 30)

			// use a token size between 2KB and 8KB

			// 5GHZ at 4KB is 20 terabytes a second
			// meaning .001% of a 5ghz CPU core is 20GB/s at 4KB write per cycle

			// if you have a write size larger than that, you have a fast data medium
			// and τ probably hasn't increased CPU speeds

			var extra = uint64(mrand.IntN(8000))

			if (extra < 2000) {
				extra = 2000;
			}

			fmt.Println("\x1b[1;31mchanging token size to: " + Bytes_to_size_string(extra) + "\x1b[0m\n\n")

			token_size.Store(extra)

		}

	}()

	var write_goroutines uint64
	for {

		if (write_goroutines == 2) {

			// have 3 writing goroutines
			break

		}

		go func() {

			for {

				var write_size = token_size.Load()

				var data_to_write = make([]byte, write_size)

				var token_pointer = rrd.QueueToken(token_queue_pointer, write_size, 0, nil)

				// write to /dev/null (io.Discard works with non Posix OS)
				io.Discard.Write(data_to_write)

				lastm_counter += write_size

				lastm_mutex.Lock()
				rrd.Update(false, time.Second, 60, rrd.Counter, rrd.GetUpdateValues(float64(lastm_counter)), &lastm_rrd)
				lastm_mutex.Unlock()

				rrd.UnqueueToken(token_queue_pointer, token_pointer)

			}

		}()

		write_goroutines += 1

	}

	select{}

}

func Bytes_to_size_string(bytes uint64) (string) {

	var size_string = ""
	var bytesf = float64(bytes)

	if (bytes > 1000 * 1000 * 1000 * 1000 * 1000) {
		size_string = fmt.Sprintf("%.2fPB", bytesf / 1000 / 1000 / 1000 / 1000 / 1000)
	} else if (bytes > 1000 * 1000 * 1000 * 1000) {
		size_string = fmt.Sprintf("%.2fTB", bytesf / 1000 / 1000 / 1000 / 1000)
	} else if (bytes > 1000 * 1000 * 1000) {
		size_string = fmt.Sprintf("%.2fGB", bytesf / 1000 / 1000 / 1000)
	} else if (bytes > 1000 * 1000) {
		size_string = fmt.Sprintf("%.2fMB", bytesf / 1000 / 1000)
	} else if (bytes > 1000) {
		size_string = fmt.Sprintf("%.2fKB", bytesf / 1000)
	} else {
		size_string = fmt.Sprintf("%.2fB", bytesf)
	}

	return size_string

}
