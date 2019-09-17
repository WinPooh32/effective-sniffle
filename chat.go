package main

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	multiaddr "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-log"
)

var logger = log.Logger("rendezvous")

func makeHandleStream(streamsMgr StreamsManager) network.StreamHandler {
	return func(stream network.Stream) {
		streamsMgr.AddStream(stream)
	}
}

func startCommunication() {
	log.SetLogLevel("*", "critical")
	// // log.SetLogLevel("dht", "critical")
	// // log.SetLogLevel("swarm2", "critical")
	// // log.SetLogLevel("relay", "critical")

	log.SetLogLevel("rendezvous", "info")

	addr := addrList{}
	addr.Set("0.0.0.0")

	config := Config{
		RendezvousString: "WOWMYSYPERSUBNET2.0",
		BootstrapPeers:   dht.DefaultBootstrapPeers,
		ProtocolID:       "/WOWMYSYPERSUBNET2/0.0.1",
		ListenAddresses:  addr,
	}
	ctx := context.Background()

	// libp2p.New constructs a new libp2p Host. Other options can be added
	// here.
	host, err := libp2p.New(ctx,
		libp2p.ListenAddrs([]multiaddr.Multiaddr(config.ListenAddresses)...),
		libp2p.NATPortMap(),
	)
	if err != nil {
		panic(err)
	}
	logger.Info("Host created. We are:", host.ID())
	logger.Info(host.Addrs())

	streamsMgr := StreamsManager{
		list:    make(StreamsMap),
		ignore:  make(IgnoreMap),
		host:    host,
		ctx:     ctx,
		protoID: protocol.ID(config.ProtocolID),
	}

	// Set a function as stream handler. This function is called when a peer
	// initiates a connection and starts a stream with this peer.
	host.SetStreamHandler(protocol.ID(config.ProtocolID), makeHandleStream(streamsMgr))

	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	logger.Info("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	// Let's connect to the bootstrap nodes first. They will tell us about the
	// other nodes in the network.
	var wg sync.WaitGroup
	for _, peerAddr := range config.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				// logger.Warning(err)
			} else {
				// logger.Info("Connection established with bootstrap node:", *peerinfo)
			}
		}()
	}
	wg.Wait()

	go func() {
		logger.Info("Searching for other peers...")
		for {

			routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
			discovery.Advertise(ctx, routingDiscovery, config.RendezvousString)

			peerChan, err := routingDiscovery.FindPeers(ctx, config.RendezvousString)
			if err != nil {
				panic(err)
			}

			tick := time.Tick(20 * time.Second)
		loop:
			for {
				select {
				case peer := <-peerChan:
					if peer.ID == host.ID() || peer.ID == "" {
						continue
					}

					streamsMgr.MakeStream(peer)
				case <-tick:
					break loop
				}
			}
		}
	}()

	select {}
}
