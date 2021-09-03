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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/hotstuff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type MsgType uint64

// messages with sub string of `Send` only used in internal communication and logs
const (
	MsgTypeNewRound     MsgType = 1
	MsgTypeSendRequest  MsgType = 2
	MsgTypeRequest      MsgType = 3
	MsgTypeSendProposal MsgType = 4
	MsgTypeProposal     MsgType = 5
	MsgTypeSendVote     MsgType = 6
	MsgTypeVote         MsgType = 7
	MsgTypeSendTimeout  MsgType = 8
	MsgTypeTimeout      MsgType = 9
	MsgTypeSendTC       MsgType = 10
	MsgTypeTC           MsgType = 11
)

func (m MsgType) String() string {
	switch m {
	case MsgTypeNewRound:
		return "MSG_NEW_ROUND"
	case MsgTypeSendRequest:
		return "MSG_SEND_REQUEST"
	case MsgTypeRequest:
		return "MSG_RECV_REQUEST"
	case MsgTypeSendProposal:
		return "MSG_SEND_PROPOSAL"
	case MsgTypeProposal:
		return "MSG_RECV_PROPOSAL"
	case MsgTypeSendVote:
		return "MSG_SEND_VOTE"
	case MsgTypeVote:
		return "MSG_RECV_VOTE"
	case MsgTypeSendTimeout:
		return "MSG_SEND_TIMEOUT"
	case MsgTypeTimeout:
		return "MSG_RECV_TIMEOUT"
	case MsgTypeSendTC:
		return "MSG_SEND_TC"
	case MsgTypeTC:
		return "MSG_RECV_TC"
	default:
		return "MSG_UNKNOWN"
	}
}

func (m MsgType) Value() uint64 {
	return uint64(m)
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
	return hotstuff.RLPHash(x)
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

func (tc *TimeoutCert) Height() *big.Int {
	if tc.View == nil {
		return common.Big0
	}
	return tc.View.Height
}

func (tc *TimeoutCert) HeightU64() uint64 {
	return tc.Height().Uint64()
}

func (tc *TimeoutCert) Round() *big.Int {
	if tc.View == nil {
		return common.Big0
	}
	return tc.View.Round
}

func (tc *TimeoutCert) RoundU64() uint64 {
	return tc.Round().Uint64()
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
