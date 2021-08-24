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
	"github.com/ethereum/go-ethereum/consensus/hotstuff"
	"github.com/ethereum/go-ethereum/event"
)

type EventSender struct {
	eventMtx *event.TypeMux
}

func NewEventSender(backend hotstuff.Backend) *EventSender {
	return &EventSender{eventMtx: backend.EventMux()}
}

func (s *EventSender) sendEvent(val interface{}) {
	s.eventMtx.Post(val)
}

// ----------------------------------------------------------------------------

// Subscribe both internal and external events
func (e *EventDrivenEngine) subscribeEvents() {
	e.events = e.backend.EventMux().Subscribe(
		// external events
		hotstuff.RequestEvent{},
		hotstuff.MessageEvent{},
		// internal events
		backlogEvent{},
	)
	e.timeoutSub = e.backend.EventMux().Subscribe(
		Timeout{},
	)
	e.finalCommittedSub = e.backend.EventMux().Subscribe(
		hotstuff.FinalCommittedEvent{},
	)
}

// Unsubscribe all events
func (e *EventDrivenEngine) unsubscribeEvents() {
	e.events.Unsubscribe()
	e.timeoutSub.Unsubscribe()
	e.finalCommittedSub.Unsubscribe()
}

func (e *EventDrivenEngine) handleEvents() {
	logger := e.logger.New("handleEvents")

	for {
		select {
		case event, ok := <-e.events.Chan():
			if !ok {
				logger.Error("Failed to receive msg Event")
				return
			}
			// A real Event arrived, process interesting content
			switch ev := event.Data.(type) {
			case hotstuff.RequestEvent:
				//e.handleRequest(&hotstuff.Request{Proposal: ev.Proposal})

			case hotstuff.MessageEvent:
				e.handleMsg(ev.Payload)

			case backlogEvent:
				e.handleCheckedMsg(ev.msg, ev.src)
			}

		case _, ok := <-e.timeoutSub.Chan():
			//logger.Trace("handle timeout Event")
			if !ok {
				logger.Error("Failed to receive timeout Event")
				return
			}
			//e.handleTimeoutMsg()

		case _, ok := <-e.finalCommittedSub.Chan():
			if !ok {
				logger.Error("Failed to receive finalCommitted Event")
				return
			}
			//e.handleFinalCommitted()
		}
	}
}

// sendEvent sends events to mux
func (e *EventDrivenEngine) sendEvent(ev interface{}) {
	e.backend.EventMux().Post(ev)
}

func (e *EventDrivenEngine) handleMsg(payload []byte) error {
	logger := e.logger.New()

	// Decode Message and check its signature
	msg := new(hotstuff.Message)
	if err := msg.FromPayload(payload, e.validateFn); err != nil {
		logger.Error("Failed to decode Message from payload", "err", err)
		return err
	}

	// Only accept Message if the address is valid
	_, src := e.valset.GetByAddress(msg.Address)
	if src == nil {
		logger.Error("Invalid address in Message", "msg", msg)
		return errInvalidSigner
	}

	// handle checked Message
	if err := e.handleCheckedMsg(msg, src); err != nil {
		return err
	}
	return nil
}

func (e *EventDrivenEngine) handleCheckedMsg(msg *hotstuff.Message, src hotstuff.Validator) (err error) {
	switch msg.Code {
	//case MsgTypeNewView:
	//	err = c.handleNewView(msg, src)
	//case MsgTypePrepare:
	//	err = c.handlePrepare(msg, src)
	//case MsgTypePrepareVote:
	//	err = c.handlePrepareVote(msg, src)
	//case MsgTypePreCommit:
	//	err = c.handlePreCommit(msg, src)
	//case MsgTypePreCommitVote:
	//	err = c.handlePreCommitVote(msg, src)
	//case MsgTypeCommit:
	//	err = c.handleCommit(msg, src)
	//case MsgTypeCommitVote:
	//	err = c.handleCommitVote(msg, src)
	default:
		err = errInvalidMessage
		e.logger.Error("msg type invalid", "unknown type", msg.Code)
	}

	if err == errFutureMessage {
		//e.storeBacklog(msg, src)
	}
	return
}

//func (c *core) handleTimeoutMsg() {
//	c.logger.Trace("handleTimeout", "state", c.currentState(), "view", c.currentView())
//	round := new(big.Int).Add(c.current.Round(), common.Big1)
//	c.startNewRound(round)
//}
