// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
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

// Package hotstuff implements the scalable hotstuff consensus algorithm.

package backend

import (
	"crypto/ecdsa"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/hotstuff"
	"github.com/ethereum/go-ethereum/consensus/hotstuff/basic"
	hsc "github.com/ethereum/go-ethereum/consensus/hotstuff/basic/core"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
)

const (
	// fetcherID is the ID indicates the block is from HotStuff engine
	fetcherID = "hotstuff"
)

// // SignerFn is a signer callback function to request a header to be signed by a
// // backing account. (Avoid import circle...)
// type SignerFn func(accounts.Account, string, []byte) ([]byte, error)

// HotStuff is the scalable hotstuff consensus engine
type backend struct {
	config *hotstuff.Config
	//db           ethdb.Database // Database to store and retrieve necessary information
	core         hsc.CoreEngine
	chain        consensus.ChainReader
	currentBlock func() *types.Block
	hasBadBlock  func(hash common.Hash) bool
	logger       log.Logger

	valset         hotstuff.ValidatorSet
	recents        *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures     *lru.ARCCache // Signatures of recent blocks to speed up mining
	recentMessages *lru.ARCCache // the cache of peer's messages
	knownMessages  *lru.ARCCache // the cache of self messages

	privateKey *ecdsa.PrivateKey
	signer     common.Address // Ethereum address of the signing key

	// The channels for hotstuff engine notifications
	sealMu            sync.Mutex
	commitCh          chan *types.Block
	proposedBlockHash common.Hash
	coreStarted       bool
	sigMu             sync.RWMutex // Protects the signer fields
	consenMu          sync.Mutex   // Ensure a round can only start after the last one has finished
	coreMu            sync.RWMutex

	// event subscription for ChainHeadEvent event
	broadcaster consensus.Broadcaster

	eventMux *event.TypeMux

	proposals map[common.Address]bool // Current list of proposals we are pushing
}

func New(config *hotstuff.Config, privateKey *ecdsa.PrivateKey, db ethdb.Database, valset hotstuff.ValidatorSet) consensus.HotStuff {
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySignatures)
	recentMessages, _ := lru.NewARC(inmemoryPeers)
	knownMessages, _ := lru.NewARC(inmemoryMessages)

	backend := &backend{
		config: config,
		//db:             db,
		logger:         log.New(),
		valset:         valset,
		commitCh:       make(chan *types.Block, 1),
		coreStarted:    false,
		eventMux:       new(event.TypeMux),
		privateKey:     privateKey,
		signatures:     signatures,
		recentMessages: recentMessages,
		knownMessages:  knownMessages,
		recents:        recents,
		proposals:      make(map[common.Address]bool),
	}

	backend.core = hsc.New(backend, config, valset)
	return backend
}

// Address implements hotstuff.Backend.Address
func (s *backend) Address() common.Address {
	return s.signer
}

// Validators implements hotstuff.Backend.Validators
func (s *backend) Validators(proposal hotstuff.Proposal) hotstuff.ValidatorSet {
	return s.getValidators()
}

// EventMux implements hotstuff.Backend.EventMux
func (s *backend) EventMux() *event.TypeMux {
	return s.eventMux
}

// Broadcast implements hotstuff.Backend.Broadcast
func (s *backend) Broadcast(valSet hotstuff.ValidatorSet, payload []byte) error {
	// send to others
	s.Gossip(valSet, payload)
	// send to self
	msg := hotstuff.MessageEvent{
		Payload: payload,
	}
	go s.EventMux().Post(msg)
	return nil
}

// Broadcast implements hotstuff.Backend.Gossip
func (s *backend) Gossip(valSet hotstuff.ValidatorSet, payload []byte) error {
	hash := hotstuff.RLPHash(payload)
	s.knownMessages.Add(hash, true)

	targets := make(map[common.Address]bool)
	for _, val := range valSet.List() { // hotstuff/validator/default.go - defaultValidator
		if val.Address() != s.Address() {
			targets[val.Address()] = true
		}
	}
	if s.broadcaster != nil && len(targets) > 0 {
		ps := s.broadcaster.FindPeers(targets)
		for addr, p := range ps {
			ms, ok := s.recentMessages.Get(addr)
			var m *lru.ARCCache
			if ok {
				m, _ = ms.(*lru.ARCCache)
				if _, k := m.Get(hash); k {
					// This peer had this event, skip it
					continue
				}
			} else {
				m, _ = lru.NewARC(inmemoryMessages)
			}

			m.Add(hash, true)
			s.recentMessages.Add(addr, m)
			go p.Send(hotstuffMsg, payload)
		}
	}
	return nil
}

// Broadcast implements hotstuff.Backend.Unicast
func (s *backend) Unicast(valSet hotstuff.ValidatorSet, payload []byte) error {
	if valSet.IsProposer(s.Address()) {
		return errInvalidProposal
	}

	hash := hotstuff.RLPHash(payload)
	s.knownMessages.Add(hash, true)

	targets := make(map[common.Address]bool)
	for _, val := range valSet.List() {
		if val.Address() != s.Address() {
			targets[val.Address()] = true
		}
	}
	if s.broadcaster != nil && len(targets) > 0 {
		ps := s.broadcaster.FindPeers(targets)
		if p, exist := ps[valSet.GetProposer().Address()]; !exist {
			return errInvalidProposal
		} else {
			go p.Send(hotstuffMsg, payload)
		}
	}
	return nil
}

// PreCommit implements hotstuff.Backend.PreCommit
func (s *backend) PreCommit(view *hotstuff.View, proposal hotstuff.Proposal, seals [][]byte) (hotstuff.Proposal, *hotstuff.QuorumCert, error) {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		s.logger.Error("Invalid proposal, %v", proposal)
		return nil, nil, errInvalidProposal
	}

	h := block.Header()
	// Append seals into extra-data
	if err := writeCommittedSeals(h, seals); err != nil {
		return nil, nil, err
	}

	// update block's header
	block = block.WithSeal(h)

	//
	qc := new(hotstuff.QuorumCert)
	qc.View = view
	qc.Hash = h.Hash()
	qc.Proposer = h.Coinbase
	qc.Extra = h.Extra

	return block, qc, nil
}

func (s *backend) Commit(proposal hotstuff.Proposal) error {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		s.logger.Error("Invalid proposal, %v", proposal)
		return errInvalidProposal
	}

	s.logger.Info("Committed", "address", s.Address(), "hash", proposal.Hash(), "number", proposal.Number().Uint64())
	// - if the proposed and committed blocks are the same, send the proposed hash
	//   to commit channel, which is being watched inside the engine.Seal() function.
	// - otherwise, we try to insert the block.
	// -- if success, the ChainHeadEvent event will be broadcasted, try to build
	//    the next block and the previous Seal() will be stopped (need to check this --- saber).
	// -- otherwise, an error will be returned and a round change event will be fired.
	if s.proposedBlockHash == block.Hash() {
		// feed block hash to Seal() and wait the Seal() result
		s.commitCh <- block
		return nil
	}

	if s.broadcaster != nil {
		s.broadcaster.Enqueue(fetcherID, block)
	}
	return nil
}

// Verify implements hotstuff.Backend.Verify
func (s *backend) Verify(proposal hotstuff.Proposal) (time.Duration, error) {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		s.logger.Error("Invalid proposal, %v", proposal)
		return 0, errInvalidProposal
	}

	// check bad block
	if s.HasBadProposal(block.Hash()) {
		return 0, core.ErrBlacklistedHash
	}

	// check block body
	txnHash := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil))
	uncleHash := types.CalcUncleHash(block.Uncles())
	if txnHash != block.Header().TxHash {
		return 0, errMismatchTxhashes
	}
	if uncleHash != nilUncleHash {
		return 0, errInvalidUncleHash
	}

	// verify the header of proposed block
	err := s.VerifyHeader(s.chain, block.Header(), false)
	if err == nil {
		return 0, nil
	} else if err == consensus.ErrFutureBlock {
		return time.Unix(int64(block.Header().Time), 0).Sub(now()), consensus.ErrFutureBlock
	}
	return 0, err
}

func (s *backend) VerifyUnsealedProposal(proposal hotstuff.Proposal) (time.Duration, error) {
	// Check if the proposal is a valid block
	block := &types.Block{}
	block, ok := proposal.(*types.Block)
	if !ok {
		s.logger.Error("Invalid proposal, %v", proposal)
		return 0, errInvalidProposal
	}

	// check bad block
	if s.HasBadProposal(block.Hash()) {
		return 0, core.ErrBlacklistedHash
	}

	// check block body
	txnHash := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil))
	uncleHash := types.CalcUncleHash(block.Uncles())
	if txnHash != block.Header().TxHash {
		return 0, errMismatchTxhashes
	}
	if uncleHash != nilUncleHash {
		return 0, errInvalidUncleHash
	}

	// verify the header of proposed block
	err := s.VerifyHeader(s.chain, block.Header(), false)
	if err == nil {
		return 0, nil
	} else if err == consensus.ErrFutureBlock {
		return time.Unix(int64(block.Header().Time), 0).Sub(now()), consensus.ErrFutureBlock
	}
	return 0, err
}

func (s *backend) VerifyQuorumCert(qc *hotstuff.QuorumCert) error {
	if qc.View.Height.Uint64() == 0 {
		return nil
	}
	extra, err := types.ExtractHotstuffExtraPayload(qc.Extra)
	if err != nil {
		return err
	}
	addr, err := basic.GetSignatureAddress(qc.Hash.Bytes(), extra.Seal)
	if err != nil {
		return err
	}
	if addr != qc.Proposer {
		return errInvalidSigner
	}

	snap := s.getValidators()
	if idx, _ := snap.GetByAddress(addr); idx < 0 {
		return errInvalidSigner
	}

	committers, err := s.signersFromCommittedSeals(qc.Hash, extra.CommittedSeal)
	if err != nil {
		return err
	}

	return s.checkValidatorQuorum(committers, snap)
}

// Sign implements hotstuff.Backend.Sign
func (s *backend) Sign(data []byte) ([]byte, error) {
	hashData := crypto.Keccak256(data)
	return crypto.Sign(hashData, s.privateKey)
}

// CheckSignature implements hotstuff.Backend.CheckSignature
func (s *backend) CheckSignature(data []byte, address common.Address, sig []byte) error {
	signer, err := hotstuff.GetSignatureAddress(data, sig)
	if err != nil {
		log.Error("Failed to get signer address", "err", err)
		return err
	}
	// Compare derived addresses
	if signer != address {
		return errInvalidSignature
	}
	return nil
}

func (s *backend) LastProposal() (hotstuff.Proposal, common.Address) {
	block := s.currentBlock()

	var proposer common.Address
	if block.Number().Cmp(common.Big0) >= 0 {
		var err error
		proposer, err = s.Author(block.Header())
		if err != nil {
			s.logger.Error("Failed to get block proposer", "err", err)
			return nil, common.Address{}
		}
	}

	// Return header only block here since we don't need block body
	return block, proposer
}

// HasProposal implements hotstuff.Backend.HashBlock
func (s *backend) HasProposal(hash common.Hash, number *big.Int) bool {
	return s.chain.GetHeader(hash, number.Uint64()) != nil
}

// GetSpeaker implements hotstuff.Backend.GetProposer
func (s *backend) GetProposer(number uint64) common.Address {
	if header := s.chain.GetHeaderByNumber(number); header != nil {
		a, _ := s.Author(header)
		return a
	}
	return common.Address{}
}

// todo:
// ParentValidators implements hotstuff.Backend.GetParentValidators
func (s *backend) ParentValidators(proposal hotstuff.Proposal) hotstuff.ValidatorSet {
	return s.getValidators()
}

// todo: use snap or reconfig validators group
func (s *backend) getValidators() hotstuff.ValidatorSet {
	return s.valset
}

func (s *backend) getValidatorsAddress() []common.Address {
	valset := s.getValidators()
	list := make([]common.Address, valset.Size())
	for i, v := range valset.List() {
		list[i] = v.Address()
	}
	return list
}

func (s *backend) HasBadProposal(hash common.Hash) bool {
	if s.hasBadBlock == nil {
		return false
	}
	return s.hasBadBlock(hash)
}
