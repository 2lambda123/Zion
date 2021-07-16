package core

import (
	"github.com/ethereum/go-ethereum/consensus/hotstuff"
)

func (c *core) handlePrepareVote(data *message, src hotstuff.Validator) error {
	logger := c.newLogger()

	var (
		vote   *hotstuff.Vote
		msgTyp = MsgTypePrepareVote
	)
	if err := data.Decode(&vote); err != nil {
		logger.Trace("Failed to decode", "type", msgTyp, "err", err)
		return errFailedDecodePrepareVote
	}
	if err := c.checkView(msgTyp, vote.View); err != nil {
		logger.Trace("Failed to check view", "msg", msgTyp, "err", err)
		return err
	}
	if err := c.checkVote(vote); err != nil {
		logger.Trace("Failed to check vote", "msg", msgTyp, "err", err)
		return err
	}
	if err := c.checkMsgToProposer(); err != nil {
		logger.Trace("Failed to check proposer", "msg", msgTyp, "err", err)
		return err
	}
	if err := c.current.AddPrepareVote(data); err != nil {
		logger.Trace("Failed to add vote", "type", msgTyp, "err", err)
		return errAddPrepareVote
	}

	logger.Trace("handlePrepareVote", "src", src.Address(), "msg view", vote.View, "vote", vote.Digest)

	if c.current.PrepareVoteSize() >= c.Q() && c.current.state < StatePrepared {
		if c.acceptPrepareVote(src, vote) {
			c.sendPreCommit()
		}
	}

	return nil
}

func (c *core) acceptPrepareVote(src hotstuff.Validator, msg *hotstuff.Vote) bool {
	logger := c.newLogger()

	seals := c.getMessageSeals(c.Q())
	newProposal, err := c.backend.PreCommit(c.current.Proposal(), seals)
	if err != nil {
		logger.Trace("Failed to assemble committed seal", "err", err)
		return false
	}

	prepareQC := proposal2QC(newProposal, c.current.Round())
	c.current.SetProposal(newProposal)
	c.current.SetPrepareQC(prepareQC)
	c.current.SetState(StatePrepared)

	logger.Trace("acceptPrepareVote", "msg", MsgTypePrepareVote, "vote digest", msg.Digest.Hex())

	return true
}

func (c *core) sendPreCommit() {
	logger := c.newLogger()

	msgTyp := MsgTypePreCommit
	msg := &MsgPreCommit{
		View:      c.currentView(),
		Proposal:  c.current.Proposal(),
		PrepareQC: c.current.PrepareQC(),
	}
	payload, err := Encode(msg)
	if err != nil {
		logger.Trace("Failed to encode", "msg", msgTyp, "err", err)
		return
	}
	c.broadcast(&message{Code: msgTyp, Msg: payload})
	logger.Trace("sendPreCommit", "msg view", msg.View, "proposal", msg.Proposal.Hash())
}

func (c *core) handlePreCommit(data *message, src hotstuff.Validator) error {
	logger := c.newLogger()

	var (
		msg    *MsgPreCommit
		msgTyp = MsgTypePreCommit
	)
	if err := data.Decode(&msg); err != nil {
		logger.Trace("Failed to check decode", "msg", msgTyp, "err", err)
		return errFailedDecodePreCommit
	}
	if err := c.checkView(MsgTypePreCommit, msg.View); err != nil {
		logger.Trace("Failed to check view", "msg", msgTyp, "err", err)
		return err
	}
	if err := c.checkMsgFromProposer(src); err != nil {
		logger.Trace("Failed to check proposer", "msg", msgTyp, "err", err)
		return err
	}
	if msg.Proposal.Hash() != msg.PrepareQC.Hash {
		logger.Trace("Failed to check msg", "msg", msgTyp, "expect prepareQC hash", msg.Proposal.Hash().Hex(), "got", msg.PrepareQC.Hash.Hex())
		return errInvalidProposal
	}
	if _, err := c.backend.Verify(msg.Proposal); err != nil {
		logger.Trace("Failed to check verify proposal", "msg", msgTyp, "err", err)
		return err
	}
	if err := c.signer.VerifyQC(msg.PrepareQC, c.valSet); err != nil {
		logger.Trace("Failed to verify prepareQC", "msg", msgTyp, "err", err)
		return errVerifyQC
	}

	logger.Trace("handlePreCommit", "src", src.Address(),"msg view", msg.View, "proposal", msg.Proposal.Hash())
	if !c.IsProposer() && c.current.state < StatePrepared {
		c.acceptPreCommit(src, msg)
	}
	c.sendPreCommitVote()

	return nil
}

func (c *core) acceptPreCommit(src hotstuff.Validator, msg *MsgPreCommit) {
	logger := c.newLogger()

	c.current.SetPrepareQC(msg.PrepareQC)
	c.current.SetProposal(msg.Proposal)
	c.current.SetState(StatePrepared)
	logger.Trace("acceptPreCommit", "msg view", msg.View, "hash", msg.PrepareQC.Hash.Hex())
}

func (c *core) sendPreCommitVote() {
	logger := c.newLogger()

	msgTyp := MsgTypePreCommitVote
	vote := c.current.Vote()
	if vote == nil {
		logger.Trace("Failed to send vote", "msg", msgTyp, "err", "current vote is nil")
		return
	}
	payload, err := Encode(vote)
	if err != nil {
		logger.Error("Failed to encode", "msg", msgTyp, "err", err)
		return
	}
	c.broadcast(&message{Code: msgTyp, Msg: payload})
	logger.Trace("sendPreCommitVote", "vote view", vote.View, "vote", vote.Digest)
}
