/*
 * Copyright (C) 2021 The Zion Authors
 * This file is part of The Zion library.
 *
 * The Zion is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The Zion is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The Zion.  If not, see <http://www.gnu.org/licenses/>.
 */

package core

import (
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/hotstuff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// hasherPool holds LegacyKeccak256 hashers for rlpHash.
var hasherPool = sync.Pool{
	New: func() interface{} { return sha3.NewLegacyKeccak256() },
}

type MsgType uint64

const (
	MsgTypeNewRound MsgType = 1
	MsgTypeProposal MsgType = 2
	MsgTypeVote     MsgType = 3
	MsgTypeTimeout  MsgType = 4
	MsgTypeQC       MsgType = 5
	MsgTypeTC       MsgType = 6
)

func (m MsgType) String() string {
	switch m {
	case MsgTypeNewRound:
		return "MSG_NEW_ROUND"
	case MsgTypeProposal:
		return "MSG_PROPOSAL"
	case MsgTypeVote:
		return "MSG_VOTE"
	case MsgTypeTimeout:
		return "MSG_TIMEOUT"
	case MsgTypeQC:
		return "MSG_QC"
	case MsgTypeTC:
		return "MSG_TC"
	default:
		return "MSG_UNKNOWN"
	}
}

func (m MsgType) Value() uint64 {
	return uint64(m)
}

type State uint64

const (
	StateAcceptRequest  State = 1
	StateAcceptProposal State = 2
	StateVoted          State = 3
)

func (s State) String() string {
	if s == StateAcceptRequest {
		return "StateAcceptRequest"
	} else if s == StateAcceptProposal {
		return "StateAcceptProposal"
	} else if s == StateVoted {
		return "StateVoted"
	} else {
		return "Unknown"
	}
}

// Cmp compares s and y and returns:
//   -1 if s is the previous state of y
//    0 if s and y are the same state
//   +1 if s is the next state of y
func (s State) Cmp(y State) int {
	if uint64(s) < uint64(y) {
		return -1
	}
	if uint64(s) > uint64(y) {
		return 1
	}
	return 0
}

type MsgProposal struct {
	Epoch     uint64
	View      *hotstuff.View
	Proposal  *types.Block
	JustifyQC *hotstuff.QuorumCert
}

func (m *MsgProposal) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{m.Epoch, m.View, m.Proposal, m.JustifyQC})
}

func (m *MsgProposal) DecodeRLP(s *rlp.Stream) error {
	var subject struct {
		Epoch     uint64
		View      *hotstuff.View
		Proposal  *types.Block
		JustifyQC *hotstuff.QuorumCert
	}

	if err := s.Decode(&subject); err != nil {
		return err
	}
	m.Epoch, m.View, m.Proposal, m.JustifyQC = subject.Epoch, subject.View, subject.Proposal, subject.JustifyQC
	return nil
}

func (m *MsgProposal) String() string {
	return fmt.Sprintf("{NewView Height: %v Round: %v Hash: %v}", m.View.Height, m.View.Round, m.Proposal.Hash())
}

type Vote struct {
	Epoch          uint64
	Hash           common.Hash
	Proposer       common.Address
	View           *hotstuff.View
	ParentHash     common.Hash
	ParentView     *hotstuff.View
	GrandHash      common.Hash
	GrandView      *hotstuff.View
	GreatGrandHash common.Hash
	GreatGrandView *hotstuff.View
}

// EncodeRLP serializes b into the Ethereum RLP format.
func (v *Vote) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{v.Epoch, v.Hash, v.Proposer, v.View, v.ParentHash, v.ParentView, v.GrandHash, v.GrandView, v.GreatGrandHash, v.GreatGrandView})
}

// DecodeRLP implements rlp.Decoder, and load the consensus fields from a RLP stream.
func (v *Vote) DecodeRLP(s *rlp.Stream) error {
	var subject struct {
		Epoch          uint64
		Hash           common.Hash
		Proposer       common.Address
		View           *hotstuff.View
		ParentHash     common.Hash
		ParentView     *hotstuff.View
		GrandHash      common.Hash
		GrandView      *hotstuff.View
		GreatGrandHash common.Hash
		GreatGrandView *hotstuff.View
	}

	if err := s.Decode(&subject); err != nil {
		return err
	}

	v.Epoch, v.Hash, v.Proposer, v.View, v.ParentHash, v.ParentView,
		v.GrandHash, v.GrandView, v.GreatGrandHash, v.GreatGrandView =
		subject.Epoch, subject.Hash, subject.Proposer, subject.View, subject.ParentHash, subject.ParentView,
		subject.GrandHash, subject.GrandView, subject.GreatGrandHash, subject.GreatGrandView
	return nil
}

func (v *Vote) String() string {
	if v.GrandView == nil {
		return fmt.Sprintf("{Epoch: %v, Hash: %v, View: %v, ParentHash: %v, ParentView: %v}", v.Epoch, v.Hash, v.View, v.ParentHash, v.ParentView)
	} else if v.GreatGrandView == nil {
		return fmt.Sprintf("{Epoch: %v, Hash: %v, View: %v, ParentHash: %v, ParentView: %v, GrandHash: %v, GrandView: %v}", v.Epoch, v.Hash, v.View, v.ParentHash, v.ParentView, v.GrandHash, v.GrandView)
	} else {
		return fmt.Sprintf("{Epoch: %v, Hash: %v, View: %v, ParentHash: %v, ParentView: %v, GrandHash: %v, GrandView: %v, GreateGrandHash: %v, GreateGrandView: %v}", v.Epoch, v.Hash, v.View, v.ParentHash, v.ParentView, v.GrandHash, v.GrandView, v.GreatGrandHash, v.GreatGrandView)
	}
}

type TimeoutEvent struct {
	Epoch  uint64
	View   *hotstuff.View
	Digest common.Hash
}

func (tm *TimeoutEvent) Hash() common.Hash {
	x := &TimeoutEvent{
		Epoch: tm.Epoch,
		View:  tm.View,
	}
	ret := make([]byte, 32)
	sha := hasherPool.Get().(crypto.KeccakState)
	defer hasherPool.Put(sha)
	sha.Reset()
	rlp.Encode(sha, x)
	sha.Read(ret[:])
	return common.BytesToHash(ret[:])
}

func (tm *TimeoutEvent) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{tm.Epoch, tm.View, tm.Digest})
}

func (tm *TimeoutEvent) DecodeRLP(s *rlp.Stream) error {
	var subject struct {
		Epoch  uint64
		View   *hotstuff.View
		Digest common.Hash
	}

	if err := s.Decode(&subject); err != nil {
		return err
	}

	tm.Epoch, tm.View, tm.Digest = subject.Epoch, subject.View, subject.Digest
	return nil
}

func (tm *TimeoutEvent) String() string {
	return fmt.Sprintf("{Epoch: %v, View: %v}", tm.Epoch, tm.View)
}

type TimeoutCert struct {
	View  *hotstuff.View
	Hash  common.Hash
	Seals [][]byte
}

// EncodeRLP serializes b into the Ethereum RLP format.
func (tc *TimeoutCert) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{tc.View, tc.Hash, tc.Seals})
}

// DecodeRLP implements rlp.Decoder, and load the consensus fields from a RLP stream.
func (tc *TimeoutCert) DecodeRLP(s *rlp.Stream) error {
	var subject struct {
		View  *hotstuff.View
		Hash  common.Hash
		Seals [][]byte
	}

	if err := s.Decode(&subject); err != nil {
		return err
	}
	tc.View, tc.Hash, tc.Seals = subject.View, subject.Hash, subject.Seals
	return nil
}

func (tc *TimeoutCert) String() string {
	return fmt.Sprintf("{TimeoutCert View: %v, Hash: %v}", tc.View, tc.Hash)
}

func (tc *TimeoutCert) Copy() *TimeoutCert {
	enc, err := rlp.EncodeToBytes(tc)
	if err != nil {
		return nil
	}
	newTc := new(TimeoutCert)
	if err := rlp.DecodeBytes(enc, &newTc); err != nil {
		return nil
	}
	return newTc
}

//
//type CertificateEvent struct {
//	Cert *hotstuff.QuorumCert
//}
//
//func (ce *CertificateEvent) EncodeRLP(w io.Writer) error {
//	return rlp.Encode(w, []interface{}{ce.Cert})
//}
//
//func (ce *CertificateEvent) DecodeRLP(s *rlp.Stream) error {
//	var subject struct {
//		Cert *hotstuff.QuorumCert
//	}
//
//	if err := s.Decode(&subject); err != nil {
//		return err
//	}
//
//	ce.Cert = subject.Cert
//	return nil
//}
//
//func (ce *CertificateEvent) String() string {
//	return fmt.Sprintf("{Hash: %v, View: %v, Proposer: %v}", ce.Cert.Hash, ce.Cert.View, ce.Cert.Proposer)
//}

type ExtraSalt struct {
	Epoch uint64
	Round *big.Int
}

func (es *ExtraSalt) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{es.Epoch, es.Round})
}

func (es *ExtraSalt) DecodeRLP(s *rlp.Stream) error {
	var subject struct {
		Epoch uint64
		Round *big.Int
	}

	if err := s.Decode(&subject); err != nil {
		return err
	}

	es.Epoch, es.Round = subject.Epoch, subject.Round
	return nil
}

func (es *ExtraSalt) String() string {
	return fmt.Sprintf("{Epoch: %v, Round: %v}", es.Epoch, es.Round)
}

type backlogEvent struct {
	src hotstuff.Validator
	msg *hotstuff.Message
}
