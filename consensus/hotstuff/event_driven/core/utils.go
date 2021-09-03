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
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/hotstuff"
	"github.com/ethereum/go-ethereum/contracts/native/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

func (e *core) checkValidatorSignature(data []byte, sig []byte) (common.Address, error) {
	return e.signer.CheckSignature(e.valset, data, sig)
}

func (e *core) newLogger() log.Logger {
	logger := e.logger.New("view", e.currentView())
	return logger
}

func (e *core) address() common.Address {
	return e.addr
}

func (e *core) isSelf(addr common.Address) bool {
	return e.addr == addr
}

func (e *core) currentView() *hotstuff.View {
	return &hotstuff.View{
		Round:  new(big.Int).Set(e.curRound),
		Height: new(big.Int).Set(e.curHeight),
	}
}

func (e *core) currentState() State {
	return e.state
}

func (e *core) checkProposer(proposer common.Address) error {
	if !e.valset.IsProposer(proposer) {
		return errNotFromProposer
	}
	return nil
}

func (e *core) checkEpoch(epoch uint64, height *big.Int) error {
	if e.epoch != epoch {
		return errInvalidHighQC
	}
	if height.Cmp(e.epochHeightStart) < 0 {
		return errInvalidEpoch
	}
	if height.Cmp(e.epochHeightEnd) > 0 {
		return errInvalidEpoch
	}
	return nil
}

func (e *core) validateProposal(proposal *types.Block) error {
	if proposal == nil {
		return errInvalidProposal
	}
	salt, _, err := extraProposal(proposal)
	if err != nil {
		return err
	}
	view := &hotstuff.View{
		Round:  salt.Round,
		Height: proposal.Number(),
	}
	if view.Cmp(e.currentView()) != 0 {
		return fmt.Errorf("expect view %v, got %v", e.currentView(), view)
	}
	if err := e.signer.VerifyHeader(proposal.Header(), e.valset, false); err != nil {
		return err
	}
	return nil
}

func (e *core) checkView(view *hotstuff.View) error {
	if e.curRound.Cmp(view.Round) != 0 || e.curHeight.Cmp(view.Height) != 0 {
		return fmt.Errorf("expect view %v, got %v", e.currentView(), view)
	}
	return nil
}

func (e *core) checkJustifyQC(proposal hotstuff.Proposal, justifyQC *hotstuff.QuorumCert) error {
	if proposal.Number().Cmp(common.Big1) == 0 {
		return nil
	}

	if justifyQC == nil {
		return fmt.Errorf("justifyQC is nil")
	}
	if justifyQC.View == nil {
		return fmt.Errorf("justifyQC view is nil")
	}
	if justifyQC.Hash == common.EmptyHash {
		return fmt.Errorf("justifyQC hash is empty")
	}
	if justifyQC.Proposer == common.EmptyAddress {
		return fmt.Errorf("justifyQC proposer is empty")
	}

	if justifyQC.View.Height.Cmp(new(big.Int).Sub(proposal.Number(), common.Big1)) != 0 {
		return fmt.Errorf("justifyQC height invalid")
	}

	if justifyQC.Hash != proposal.ParentHash() {
		return fmt.Errorf("justifyQC hash extendship invalid")
	}

	vs := e.valset.Copy()
	vs.CalcProposerByIndex(justifyQC.View.Round.Uint64())
	proposer := vs.GetProposer().Address()
	if proposer != justifyQC.Proposer {
		return fmt.Errorf("justifyQC proposer expect %v got %v", proposer, justifyQC.Proposer)
	}

	return nil
}

func (e *core) compareQC(expect, src *hotstuff.QuorumCert) error {
	if expect.Hash != src.Hash {
		return fmt.Errorf("qc hash expect %v, got %v", expect.Hash, src.Hash)
	}
	if expect.View.Cmp(src.View) != 0 {
		return fmt.Errorf("qc view expect %v, got %v", expect.View, src.View)
	}
	if expect.Proposer != src.Proposer {
		return fmt.Errorf("qc proposer expect %v, got %v", expect.Proposer, src.Proposer)
	}
	if !bytes.Equal(expect.Extra, src.Extra) {
		return fmt.Errorf("qc extra not same")
	}
	return nil
}

// vote to highQC round + 1
func (e *core) checkVote(vote *Vote) error {
	if vote.View == nil {
		return fmt.Errorf("vote view is nil")
	}
	if vote.Hash == utils.EmptyHash {
		return fmt.Errorf("vote hash is empty")
	}
	if vote.ParentView == nil {
		return fmt.Errorf("vote parent view is nil")
	}
	if vote.ParentHash == utils.EmptyHash {
		return fmt.Errorf("vote parent hash is empty")
	}

	// vote view MUST be highQC view
	highQC := e.blkPool.GetHighQC()
	if new(big.Int).Sub(vote.View.Height, highQC.View.Height).Cmp(common.Big1) != 0 &&
		new(big.Int).Sub(vote.View.Round, highQC.View.Round).Cmp(common.Big1) != 0 {
		return errInvalidVote
	}
	return nil
}

func (e *core) getVoteSeals(hash common.Hash, n int) [][]byte {
	seals := make([][]byte, n)
	for i, data := range e.messages.Votes(hash) {
		if i < n {
			seals[i] = data.CommittedSeal
		}
	}
	return seals
}

func (e *core) getTimeoutSeals(round uint64, n int) [][]byte {
	seals := make([][]byte, n)
	for i, data := range e.messages.Timeouts(round) {
		if i < n {
			seals[i] = data.CommittedSeal
		}
	}
	return seals
}

func (e *core) Q() int {
	return e.valset.Q()
}

func (e *core) chain2Height() *big.Int {
	return new(big.Int).Add(e.epochHeightStart, common.Big2)
}

func (e *core) chain3Height() *big.Int {
	return new(big.Int).Add(e.epochHeightStart, common.Big3)
}

func (e *core) generateTimeoutEvent() *TimeoutEvent {
	tm := &TimeoutEvent{
		Epoch: e.epoch,
		View:  e.currentView(),
	}
	tm.Digest = tm.Hash()
	return tm
}

func (e *core) aggregateQC(vote *Vote, size int) (*hotstuff.QuorumCert, *types.Block, error) {
	proposal := e.blkPool.GetBlockAndCheckHeight(vote.Hash, vote.View.Height)
	if proposal == nil {
		return nil, nil, fmt.Errorf("last proposal %v not exist", vote.Hash)
	}

	seals := e.getVoteSeals(vote.Hash, size)
	sealedProposal, err := e.backend.PreCommit(proposal, seals)
	if err != nil {
		return nil, nil, err
	}

	sealedBlock, ok := sealedProposal.(*types.Block)
	if !ok {
		return nil, nil, errProposalConvert
	}

	extra := sealedBlock.Header().Extra
	qc := &hotstuff.QuorumCert{
		View:     vote.View,
		Hash:     sealedProposal.Hash(),
		Proposer: sealedProposal.Coinbase(),
		Extra:    extra,
	}

	return qc, proposal, nil
}

func (e *core) aggregateTC(event *TimeoutEvent, size int) *TimeoutCert {
	seals := e.getTimeoutSeals(event.View.Round.Uint64(), size)
	tc := &TimeoutCert{
		View:  event.View,
		Hash:  common.Hash{},
		Seals: seals,
	}
	return tc
}

func (e *core) nextValset() hotstuff.ValidatorSet {
	vs := e.valset.Copy()
	vs.CalcProposerByIndex(e.curRound.Uint64() + 1)
	return vs
}

func (e *core) nextProposer() common.Address {
	vs := e.valset.Copy()
	vs.CalcProposerByIndex(e.curRound.Uint64() + 1)
	proposer := vs.GetProposer()
	return proposer.Address()
}

func Encode(val interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(val)
}

func copyNum(src *big.Int) *big.Int {
	return new(big.Int).Set(src)
}

func isTC(qc *hotstuff.QuorumCert) bool {
	if qc.Hash == utils.EmptyHash {
		return true
	}
	return false
}

func sub1(num *big.Int) *big.Int {
	return new(big.Int).Sub(num, common.Big1)
}
