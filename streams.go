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

	list    StreamsMap
	ctx     context.Context
	host    host.Host
	protoID protocol.ID
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
		// fmt.Println(err)
		// failed to dial
		return
	}

	ss.AddStream(stream)
}

func (ss *StreamsManager) AddStream(stream network.Stream) {
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	go readData(rw)
	go writeData(ss)

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

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			return
		}

		if str == "" {
			return
		}
		if str != "\n" {
			fmt.Printf("> %s", str)
		}
	}
}

func writeData(ss *StreamsManager) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			return
		}

		ss.WriteToAll([]byte(sendData))
	}
}
