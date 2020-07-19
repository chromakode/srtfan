package main

import (
	"context"
	"flag"
	"log"
	"net"
	"strconv"
	"sync"
	"syscall"

	"github.com/openfresh/gosrt/srt"
	"github.com/openfresh/gosrt/srtapi"
)

var (
	outConns  = map[int]net.Conn{}
	mux       = sync.Mutex{}
	nextOutID = 0
)

func listenSinks(srtCtx context.Context, sinkAddress string) {
	listener, err := srt.ListenContext(srtCtx, "srt", sinkAddress)
	if err != nil {
		log.Fatalf("failed to listen for sink: %v", err)
	}
	log.Printf("listening for sinks: %v", listener.Addr().String())

	for {
		outConn, err := listener.Accept()
		if err != nil {
			log.Printf("error accepting sink connection: %v", err)
			continue
		}

		log.Printf("sink connected: %v", outConn.RemoteAddr().String())
		mux.Lock()
		connID := nextOutID
		nextOutID++
		outConns[connID] = outConn
		mux.Unlock()
	}
}

func listenSrc(srtCtx context.Context, srcAddress string) {
	for {
		listener, err := srt.ListenContext(srtCtx, "srt", srcAddress)
		if err != nil {
			log.Fatalf("failed to listen for src: %v", err)
		}
		log.Printf("listening for sources: %v", listener.Addr().String())

		inConn, err := listener.Accept()
		listener.Close()
		if err != nil {
			log.Printf("error accepting src connection: %v", err)
			continue
		}

		log.Printf("source connected: %v", inConn.RemoteAddr().String())
		for {
			b := make([]byte, 1316)
			n, err := inConn.Read(b)
			if err != nil {
				log.Printf("source disconnected: %v", inConn.RemoteAddr().String())
				inConn.Close()
				break
			}

			mux.Lock()
			for connID, outConn := range outConns {
				_, err := outConn.Write(b[:n])
				if err != nil {
					log.Printf("sink disconnected: %v", outConn.RemoteAddr().String())
					delete(outConns, connID)
				}
			}
			mux.Unlock()
		}
	}
}

func main() {
	srcAddress := flag.String("src-address", ":5000", "Address to listen for source connection")
	srcLatency := flag.Int("src-latency", 500, "SRT source latency parameter")
	srcPassphrase := flag.String("src-passphrase", "", "SRT source passphrase parameter")
	sinkAddress := flag.String("sink-address", ":5001", "Address to listen for sink connections")
	sinkLatency := flag.Int("sink-latency", 500, "SRT sink latency parameter")
	sinkPassphrase := flag.String("sink-passphrase", "", "SRT sink passphrase parameter")
	flag.Parse()

	srcSRTCtx := srt.WithOptions(context.Background(), srt.Options("latency", strconv.Itoa(*srcLatency)))
	if *srcPassphrase != "" {
		if len(*srcPassphrase) < 10 {
			log.Fatal("-src-passphrase: must be at least 10 characters long")
		}
		srcSRTCtx = srt.WithListenCallback(srcSRTCtx, func(ns int, hsversion int, peeraddr syscall.Sockaddr, streamID string) int {
			if err := srtapi.SetsockflagString(ns, int(srtapi.OptionPassphrase), *srcPassphrase); err != nil {
				log.Fatalf("failed to set src socket passphrase: %v", err)
			}
			return 0
		})
	}
	sinkSRTCtx := srt.WithOptions(context.Background(), srt.Options("latency", strconv.Itoa(*sinkLatency)))
	if *sinkPassphrase != "" {
		if len(*sinkPassphrase) < 10 {
			log.Fatal("-sink-passphrase: must be at least 10 characters long")
		}
		sinkSRTCtx = srt.WithListenCallback(sinkSRTCtx, func(ns int, hsversion int, peeraddr syscall.Sockaddr, streamID string) int {
			if err := srtapi.SetsockflagString(ns, int(srtapi.OptionPassphrase), *sinkPassphrase); err != nil {
				log.Fatalf("failed to set sink socket passphrase: %v", err)
			}

			// To support older SRT client versions (e.g. VLC) this is necessary for the passphrase handshake to occur.
			// See: https://github.com/Haivision/srt/blob/9654eb65e694128988569ddf7f45d8bccb78f267/docs/API.md
			srtapi.SetsockflagBool(ns, int(srtapi.OptionSender), true)
			return 0
		})
	}

	go listenSinks(sinkSRTCtx, *sinkAddress)
	listenSrc(srcSRTCtx, *srcAddress)
}
