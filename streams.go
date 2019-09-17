package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"golang.org/x/net/context"
)

type StreamsMap map[peer.ID]ChatStream
type IgnoreMap map[peer.ID]struct{}

type ChatStream struct {
	network.Stream
	RWBuffer *bufio.ReadWriter
}

func (cs *ChatStream) Write(data []byte) {
	_, err := cs.RWBuffer.Write(data)
	if err != nil {
		return
	}
	err = cs.RWBuffer.Flush()
	if err != nil {
		return
	}
}

// StreamsManager manages connections streams
type StreamsManager struct {
	sync.Mutex

	ignore  IgnoreMap
	list    StreamsMap
	ctx     context.Context
	host    host.Host
	protoID protocol.ID
}

func (ss *StreamsManager) Ignore(id peer.ID) {
	ss.ignore[id] = struct{}{}
}

func (ss *StreamsManager) IsIgnored(id peer.ID) bool {
	_, ok := ss.ignore[id]
	return ok
}

func (ss *StreamsManager) CloseByPeer(id peer.ID) {
	if stream, ok := ss.list[id]; ok {
		if err := stream.Close(); err != nil {
			fmt.Println(err)
		}

		ss.Lock()
		delete(ss.list, id)
		ss.Unlock()
	}
}

func (ss *StreamsManager) MakeStream(peer peer.AddrInfo) {
	if _, ok := ss.list[peer.ID]; ok {
		return
	}

	stream, err := ss.host.NewStream(ss.ctx, peer.ID, ss.protoID)
	if err != nil {
		// failed to dial
		ss.Ignore(peer.ID)
		return
	}

	ss.AddStream(stream)
}

func (ss *StreamsManager) AddStream(stream network.Stream) {
	peerLong := stream.Conn().RemotePeer()
	peerShort := peerLong[len(peerLong)-5:]

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw, peerShort)

	if len(ss.list) == 0 {
		hostLong := ss.host.ID()
		hostShort := hostLong[len(hostLong)-5:]
		go writeData(ss, hostShort)
	}

	ss.Lock()
	ss.list[stream.Conn().RemotePeer()] = ChatStream{
		Stream:   stream,
		RWBuffer: rw,
	}
	ss.Unlock()
}

func (ss *StreamsManager) WriteToAll(data []byte) {
	for _, stream := range ss.list {
		stream.RWBuffer.Write(data)
		err := stream.RWBuffer.Flush()
		if err != nil {
			ss.CloseByPeer(stream.Conn().RemotePeer())
		}
	}
}

func readData(rw *bufio.ReadWriter, peerShort peer.ID) {

	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			return
		}

		if str == "" {
			return
		}
		if str != "\n" {
			fmt.Printf("%s> %s", peerShort, str)
		}
	}
}

func writeData(ss *StreamsManager, peerShort peer.ID) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s (ME)> ", peerShort)
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			return
		}

		ss.WriteToAll([]byte(sendData))
	}
}
