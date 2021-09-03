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

import "github.com/ethereum/go-ethereum/consensus/hotstuff"

func (e *core) sendVote() error {
	logger := e.newLogger()

	view := e.currentView()
	justifyQC := e.smr.HighQC()
	proposal := e.smr.Proposal()

	// make vote and send it to next proposer
	vote, err := e.makeVote(proposal.Hash(), proposal.Coinbase(), view, justifyQC)
	if err != nil {
		logger.Trace("Failed to make vote", "err", err)
		return err
	}

	logger.Trace("Send Vote", "to", e.nextProposer(), "hash", vote.Hash)

	e.increaseLastVoteRound(view.Round)
	e.encodeAndBroadcast(MsgTypeVote, vote)

	return nil
}

// handleVote validate vote message and try to assemble qc
func (e *core) handleVote(src hotstuff.Validator, data *hotstuff.Message) error {
	var (
		vote   *Vote
		msgTyp = MsgTypeVote
	)

	logger := e.newLogger()
	if err := data.Decode(&vote); err != nil {
		logger.Trace("Failed to decode", "msg", msgTyp, "from", src.Address(), "err", err)
		return errFailedDecodeNewView
	}

	logger.Trace("Accept Vote", "msg", msgTyp, "from", src.Address(), "hash", vote.Hash, "vote view", vote.View)

	if err := e.checkVote(vote); err != nil {
		logger.Trace("Failed to check vote", "msg", msgTyp, "from", src.Address(), "err", err)
		return err
	}
	if err := e.checkEpoch(vote.Epoch, vote.View.Height); err != nil {
		logger.Trace("Failed to check epoch", "msg", msgTyp, "from", src.Address(), "err", err)
		return err
	}
	if err := e.validateVote(vote); err != nil {
		logger.Trace("Failed to validate vote", "msg", msgTyp, "from", src.Address(), "err", err)
		return err
	}
	if err := e.messages.AddVote(vote.Hash, data); err != nil {
		logger.Trace("Failed to add vote", "msg", msgTyp, "from", src.Address(), "err", err)
		return err
	}

	size := e.messages.VoteSize(vote.Hash)
	if size != e.Q() {
		return nil
	}

	qc, proposal, err := e.aggregateQC(vote, size)
	if err != nil {
		logger.Trace("Failed to aggregate qc", "msg", msgTyp, "err", err)
		return err
	}
	logger.Trace("Aggregate QC", "msg", msgTyp, "qc", qc.Hash, "view", qc.View)

	if err := e.blkPool.AddBlock(proposal, vote.View.Round); err != nil {
		logger.Trace("Failed to insert block into block pool", "msg", msgTyp, "err", err)
		return err
	}
	e.smr.SetHighQC(qc)
	highQC := e.smr.HighQC()

	if err := e.advanceRoundByQC(highQC, false); err != nil {
		logger.Trace("Failed to advance round", "msg", msgTyp, "err", err)
		return err
	}

	return nil
}
