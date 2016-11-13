package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const ianaProtocolICMP = 1

const ianaProtocolIPv6ICMP = 58

const ianaICMPTypeEcho = 8

const metricsPrefix = "network_health"

const pingPrefix = "NETWORK_HEALTH_PING"

// globals
var cfg Configuration

func sendPing(conn *icmp.PacketConn, rspChannel chan *pingPending, host string, rqID int) error {
	start := time.Now()

	dst, err := resolveHost(conn, ianaProtocolICMP, host)
	if err != nil {
		return err
	}

	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1 << uint(rqID),
			Data: []byte(fmt.Sprintf("%s%d", pingPrefix, rqID)),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}
	if n, err := conn.WriteTo(wb, dst); err != nil {
		return err
	} else if n != len(wb) {
		return fmt.Errorf("got %v; want %v", n, len(wb))
	}

	r := &pingPending{
		id:    rqID,
		host:  host,
		start: start.UnixNano(),
		resp:  make(chan bool)}
	rspChannel <- r
	<-r.resp // wait until it has effectively been queued
	return nil
}

func parseResults(cfg Configuration, conn *icmp.PacketConn, rspChannel chan *pingResponse) error {
	rb := make([]byte, 1500)
	if err := conn.SetReadDeadline(time.Now().Add(
		time.Duration(cfg.TimeoutSeconds) * time.Second)); err != nil {
		return err
	}
	n, _, err := conn.ReadFrom(rb)
	if err != nil {
		return err
	}
	rm, err := icmp.ParseMessage(ianaProtocolICMP, rb[:n])
	if err != nil {
		return err
	}
	switch rm.Type {
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		bodyBytes, err := rm.Body.Marshal(ianaProtocolICMP)
		if err != nil {
			return err
		}
		message := bodyBytes[4:] // skip the first 2 bytes
		end := time.Now()
		id, err := strconv.Atoi(strings.TrimPrefix(string(message), pingPrefix))
		if err != nil {
			return err // This can happen if we receive packets not produced by this app
		}

		r := &pingResponse{
			id: id,
			t:  end.UnixNano()}
		rspChannel <- r
		if err != nil {
			return err
		}
	}
	return nil
}

func showUsage() {
	fmt.Printf("Usage : network_health_exporter --config <path-to-conf.json>\n")
}

func scheduleTimeouts(timeouts chan bool) {
	go func() {
		for {
			time.Sleep(time.Second)
			timeouts <- true
		}
	}()
}

func startStateManager(cfg Configuration, reads chan *readOp, pendings chan *pingPending, responses chan *pingResponse) {
	timeouts := make(chan bool)
	timeoutNanos := (time.Duration(cfg.TimeoutSeconds) * time.Second).Nanoseconds()

	go func() {
		var state = make(map[string]*hostResults)
		var inProgress = make(map[int]*pingPending)

		for {
			select {
			case read := <-reads:
				read.resp <- state
			case r := <-responses:
				pending, ok := inProgress[r.id]
				if ok {
					elapsed := int(r.t - pending.start)
					currentHost, ok := state[pending.host]
					if ok {
						currentHost.lastElapsed = elapsed
						currentHost.completed++
					} else {
						newHost := &hostResults{
							host:        pending.host,
							lastElapsed: elapsed,
							completed:   1,
							timeouts:    0}
						state[pending.host] = newHost
					}
					delete(inProgress, r.id)
				}
				// optionally respond to the rq
				if r.resp != nil {
					r.resp <- true
				}

			case <-timeouts:
				now := time.Now().UnixNano()
				for k, v := range inProgress {
					if v.start+timeoutNanos < now {
						delete(inProgress, k)
						hostResult, ok := state[v.host]
						if ok {
							hostResult.timeouts++
						} else {
							newHost := &hostResults{
								host:        v.host,
								lastElapsed: int(timeoutNanos),
								completed:   0,
								timeouts:    1}
							state[v.host] = newHost
						}
					}
				}

			case pending := <-pendings:
				inProgress[pending.id] = pending
				pending.resp <- true
			}
		}
	}()

	scheduleTimeouts(timeouts)
}

func startParseResults(cfg Configuration, conn *icmp.PacketConn, resultsChannel chan *pingResponse) {
	go func() {
		for {
			parseResults(cfg, conn, resultsChannel)
		}
	}()
}

func startScheduling(cfg Configuration, conn *icmp.PacketConn, pendings chan *pingPending) {
	go func() {
		i := 0
		for {
			fmt.Println("Scheduling a new batch of pings...")
			for _, host := range cfg.Targets {
				go sendPing(conn, pendings, host, i)
				i++
			}
			time.Sleep(time.Second * time.Duration(cfg.IntervalSeconds))
		}
	}()
}

func run(cfg Configuration) {
	// channel to read the current state of the results
	reads := make(chan *readOp)
	// channel for in progress ping operations
	pendings := make(chan *pingPending)
	// channel to receive ping responses
	responses := make(chan *pingResponse)

	startStateManager(cfg, reads, pendings, responses)

	conn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		fmt.Println("Error while opening listening conn", err)
		os.Exit(1)
	}

	startParseResults(cfg, conn, responses)

	startScheduling(cfg, conn, pendings)

	startServer(cfg, reads)
}

func main() {

	args := os.Args[1:]
	if len(args) <= 1 || args[0] != "--config" {
		showUsage()
	}
	confLoc := args[1]
	cfg := ReadConfig(confLoc)

	run(cfg)
}
