package main

import (
	"bufio"
	"fmt"
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
	ctrl    uiControl
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

		peerLong := stream.Conn().RemotePeer()
		peerShort := peerLong[len(peerLong)-5:]
		ss.ctrl.DelUser <- peerShort.Pretty()
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

	go readData(ss, rw, peerShort)

	ss.Lock()
	ss.list[stream.Conn().RemotePeer()] = ChatStream{
		Stream:   stream,
		RWBuffer: rw,
	}
	ss.Unlock()

	ss.ctrl.AddUser <- fmt.Sprint(peerShort)
}

func (ss *StreamsManager) WriteToAll(data []byte) {
	for _, stream := range ss.list {
		fmt.Println("WRITE TO ALL")

		stream.RWBuffer.Write(data)
		err := stream.RWBuffer.Flush()
		if err != nil {
			ss.CloseByPeer(stream.Conn().RemotePeer())
		}
	}
}

func readData(ss *StreamsManager, rw *bufio.ReadWriter, peerShort peer.ID) {
	for {
		fmt.Println("READ")

		str, err := rw.ReadString('\n')
		if err != nil {
			return
		}

		if str == "" {
			return
		}

		if str != "\n" {
			ss.ctrl.AddMessage <- str[:len(str)-1]
		}

	}
}

func writeData(ss *StreamsManager, peerShort peer.ID) {
	for {
		sendData := <-ss.ctrl.SubmitMessage
		ss.WriteToAll([]byte(fmt.Sprint(sendData, "\n")))
		fmt.Println("WRITE")
	}
}
