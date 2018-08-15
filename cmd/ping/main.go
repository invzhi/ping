package main

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/invzhi/ping"
)

const (
	tickTime  = 500 * time.Millisecond
	sleepTime = 500 * time.Millisecond
)

func main() {
	if len(os.Args) == 0 {
		fmt.Printf("Usage: sudo %s hostname\n", os.Args[0])
		os.Exit(1)
	}

	hostname := os.Args[1]
	ipAddr, err := net.ResolveIPAddr("ip4", hostname)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("PING %v (%v).\n", hostname, ipAddr)

	var (
		transmitted int
		received    int
	)
	// icmpSeq, received := 0, 0
	reply, timeout := make(chan string), make(chan string)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	startTime := time.Now()
	var endTime int

	go func() {
		ticker := time.NewTicker(tickTime)

	ping:
		for {
			select {
			case <-quit:
				signal.Stop(quit)
				ticker.Stop()
				endTime = int(time.Since(startTime).Seconds() * 1000)
				break ping
			case <-ticker.C:
				go ping.Do(ipAddr, transmitted, reply, timeout)
				transmitted++
			}
		}
		time.Sleep(sleepTime * 500)
		close(reply)
	}()

	for info := range reply {
		received++
		fmt.Println(info)
	}

	if len(ping.RTTs) > 0 {
		loss := transmitted - received

		fmt.Printf("\n--- %v ping statistics ---\n", hostname)
		fmt.Printf("%v packets transmitted, %v received, %v%% packet loss, time %vms\n",
			transmitted, received, loss, endTime,
		)
		fmt.Printf("rtt min/avg/max/mdev = %v/%.3f/%v/%.3f ms\n",
			minRTT(ping.RTTs),
			avgRTT(ping.RTTs),
			maxRTT(ping.RTTs),
			mdevRTT(ping.RTTs),
		)
	}
}

func minRTT(rtts []int) int {
	min := rtts[0]
	for _, rtt := range rtts {
		if rtt < min {
			min = rtt
		}
	}
	return min
}

func maxRTT(rtts []int) int {
	max := rtts[0]
	for _, rtt := range rtts {
		if rtt > max {
			max = rtt
		}
	}
	return max
}

func avgRTT(rtts []int) float64 {
	var avg float64
	for _, rtt := range rtts {
		avg += float64(rtt)
	}
	return avg / float64(len(rtts))
}

func mdevRTT(rtts []int) float64 {
	var mdev float64
	for _, rtt := range rtts {
		mdev += float64(rtt * rtt)
	}
	avg := avgRTT(rtts)
	mdev = mdev/float64(len(rtts)) - avg*avg
	return math.Sqrt(mdev)
}
