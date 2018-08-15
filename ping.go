package ping

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var RTTs []int

const (
	ProtocolICMP = 1
)

func Do(ipAddr *net.IPAddr, icmpSeq int, reply, timeout chan string) {
	packetConn, err := icmp.ListenPacket("ip4:icmp", "")
	if err != nil {
		log.Fatal(err)
	}
	defer packetConn.Close()

	icmpEchoRequset := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  icmpSeq,
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
	RTTs = append(RTTs, rtt)
	icmpEchoReply, err := icmp.ParseMessage(ProtocolICMP, buffer[:n])
	if err != nil {
		log.Fatal(err)
	}

	switch icmpEchoReply.Type {
	case ipv4.ICMPTypeEchoReply:
		reply <- fmt.Sprintf("%v bytes from %v: icmp_seq=%v time=%v ms", n+20, ipAddr, icmpSeq, rtt)
	case ipv4.ICMPTypeDestinationUnreachable:
		reply <- fmt.Sprintf("%v is unreachable", ipAddr)
	default:
		reply <- fmt.Sprintf("got %+v\n", icmpEchoReply)
	}
}
