package main

import (
	"os"
	"fmt"
	"net"
	"log"
	"time"
	"flag"
	"math"
	"syscall"
	"os/signal"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/icmp"
)

var rtts []int

func main() {
	flag.Parse()
	hostname := flag.Arg(0)
	if len(hostname) == 0 {
		fmt.Printf("Usage: sudo %s hostname\n", os.Args[0])
		os.Exit(1)
	}

	ipAddr, err := net.ResolveIPAddr("ip4", hostname)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("PING %v (%v).\n", hostname, ipAddr)

	icmpSeq, received, loss := 0, 0, 0
	reply, timeout := make(chan string), make(chan string)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	startTime := time.Now()
	var endTime int

	go func() {
		tick := time.Tick(time.Millisecond * 500)

		ping:
		for {
			select {
				case <- quit:
					signal.Stop(quit)
					endTime = int(time.Since(startTime).Seconds() * 1000)
					break ping
				case <- tick:
					go ping(ipAddr, icmpSeq, reply, timeout)
					icmpSeq++
			}
		}
		time.Sleep(time.Millisecond * 500)
		close(reply)
	}()

	for info := range reply {
		received++
		fmt.Println(info)
	}

	if len(rtts) > 0 {
		loss = icmpSeq - received

		fmt.Printf("\n--- %v ping statistics ---\n", hostname)
		fmt.Printf("%v packets transmitted, %v received, %v%% packet loss, time %vms\n",
			icmpSeq, received, loss, endTime,
		)
		fmt.Printf("rtt min/avg/max/mdev = %v/%.3f/%v/%.3f ms\n",
			minRTT(rtts),
			avgRTT(rtts),
			maxRTT(rtts),
			mdevRTT(rtts),
		)
	}
}

func ping(ipAddr *net.IPAddr, icmpSeq int, reply chan string, timeout chan string) {
	packetConn, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		log.Fatal(err)
	}
	defer packetConn.Close()

	icmpEchoRequset := icmp.Message {
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff,
			Seq: icmpSeq,
			Data: []byte("HELLO-PING-PING"),
		},
	}
	icmpBytes, err := icmpEchoRequset.Marshal(nil)
	if err != nil {
		log.Fatal(err)
	}

	_, err = packetConn.WriteTo(icmpBytes, ipAddr)
	if err != nil {
		log.Fatal(err)
	}
	sendTime := time.Now()

	buffer := make([]byte, 1500)
	packetConn.SetReadDeadline(sendTime.Add(time.Millisecond * 400))
	n, _, err := packetConn.ReadFrom(buffer)
	if err != nil {
		if neterr, ok := err.(*net.OpError); ok {
			if neterr.Timeout() {
				timeout <- "ping timeout"
				return
			}
		} else {
			log.Fatal(err)
		}
	}
	rtt := int(time.Since(sendTime).Seconds() * 1000)
	rtts = append(rtts, rtt)
	icmpEchoReply, err := icmp.ParseMessage(1, buffer[:n])
	if err != nil {
		log.Fatal(err)
	}

	switch icmpEchoReply.Type {
	case ipv4.ICMPTypeEchoReply:
		reply <- fmt.Sprintf("%v bytes from %v: icmp_seq=%v time=%v ms", n + 20, ipAddr, icmpSeq, rtt)
	case ipv4.ICMPTypeDestinationUnreachable:
		reply <- fmt.Sprintf("%v is unreachable", ipAddr)
	default:
		reply <- fmt.Sprintf("got %+v\n", icmpEchoReply)
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
		mdev += float64(rtt*rtt)
	}
	avg := avgRTT(rtts)
	mdev = mdev/float64(len(rtts)) - avg*avg
	return math.Sqrt(mdev)
}