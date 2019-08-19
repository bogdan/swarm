//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package swap

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethersphere/swarm/log"
	"github.com/ethersphere/swarm/p2p/protocols"
)

// ErrEmptyAddressInSignature is used when the empty address is used for the chequebook in the handshake
var ErrEmptyAddressInSignature = errors.New("empty address in handshake")

// Spec is the swap protocol specification
var Spec = &protocols.Spec{
	Name:       "swap",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		HandshakeMsg{},
		EmitChequeMsg{},
		ErrorMsg{},
	},
}

// Protocols is a node.Service interface method
func (s *Swap) Protocols() []p2p.Protocol {
	return []p2p.Protocol{
		{
			Name:    Spec.Name,
			Version: Spec.Version,
			Length:  Spec.Length(),
			Run:     s.run,
		},
	}
}

// APIs is a node.Service interface method
func (s *Swap) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "swap",
			Version:   "1.0",
			Service:   s.api,
			Public:    false,
		},
	}
}

// Start is a node.Service interface method
func (s *Swap) Start(server *p2p.Server) error {
	log.Info("Swap service started")
	return nil
}

// Stop is a node.Service interface method
func (s *Swap) Stop() error {
	return nil
}

// verifyHandshake verifies the chequebook address transmitted in the swap handshake
func (s *Swap) verifyHandshake(msg interface{}) error {
	handshake, ok := msg.(*HandshakeMsg)
	if !ok || (handshake.ContractAddress == common.Address{}) {
		return ErrEmptyAddressInSignature
	}

	return s.verifyContract(context.TODO(), handshake.ContractAddress)
}

// run is the actual swap protocol run method
func (s *Swap) run(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	protoPeer := protocols.NewPeer(p, rw, Spec)

	answer, err := protoPeer.Handshake(context.TODO(), &HandshakeMsg{
		ContractAddress: s.owner.Contract,
	}, s.verifyHandshake)
	if err != nil {
		return err
	}

	beneficiary, err := s.getContractOwner(context.TODO(), answer.(*HandshakeMsg).ContractAddress)
	if err != nil {
		return err
	}

	swapPeer := NewPeer(protoPeer, s, s.backend, beneficiary, answer.(*HandshakeMsg).ContractAddress)
	s.addPeer(swapPeer)
	defer s.removePeer(swapPeer)

	s.logBalance(protoPeer)

	return swapPeer.Run(swapPeer.handleMsg)
}

func (s *Swap) removePeer(p *Peer) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.peers, p.ID())
}

func (s *Swap) addPeer(p *Peer) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.peers[p.ID()] = p
}

func (s *Swap) getPeer(id enode.ID) *Peer {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.peers[id]
}

// PublicAPI would be the public API accessor for protocol methods
type PublicAPI struct {
}