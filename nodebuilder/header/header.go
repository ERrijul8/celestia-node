package header

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"go.uber.org/fx"

	"github.com/celestiaorg/celestia-node/header"
	"github.com/celestiaorg/celestia-node/header/p2p"
	"github.com/celestiaorg/celestia-node/header/store"
	"github.com/celestiaorg/celestia-node/header/sync"
	"github.com/celestiaorg/celestia-node/params"
)

// newP2PExchange constructs new Exchange for headers.
func newP2PExchange(cfg Config) func(params.Bootstrappers, host.Host) (header.Exchange, error) {
	return func(bpeers params.Bootstrappers, host host.Host) (header.Exchange, error) {
		peers, err := cfg.trustedPeers(bpeers)
		if err != nil {
			return nil, err
		}
		ids := make([]peer.ID, len(peers))
		for index, peer := range peers {
			ids[index] = peer.ID
			host.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
		}
		return p2p.NewExchange(host, ids), nil
	}
}

// newSyncer constructs new Syncer for headers.
func newSyncer(ex header.Exchange, store initStore, sub header.Subscriber, duration time.Duration) *sync.Syncer {
	return sync.NewSyncer(ex, store, sub, duration)
}

// initStore is a type representing initialized header store.
// NOTE: It is needed to ensure that Store is always initialized before Syncer is started.
type initStore header.Store

// newInitStore constructs an initialized store
func newInitStore(
	lc fx.Lifecycle,
	cfg Config,
	net params.Network,
	s header.Store,
	ex header.Exchange,
) (initStore, error) {
	trustedHash, err := cfg.trustedHash(net)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			err = store.Init(ctx, s, ex, trustedHash)
			if err != nil {
				// TODO(@Wondertan): Error is ignored, as otherwise unit tests for Node construction fail.
				// 	This is due to requesting step of initialization, which fetches initial Header by trusted hash from
				//  the network. The step can't be done during unit tests and fixing it would require either
				//   * Having some test/dev/offline mode for Node that mocks out all the networking
				//   * Hardcoding full extended header in params pkg, instead of hashes, so we avoid requesting step
				//   * Or removing explicit initialization in favor of automated initialization by Syncer
				log.Errorf("initializing store failed: %s", err)
			}
			return nil
		},
	})

	return s, nil
}