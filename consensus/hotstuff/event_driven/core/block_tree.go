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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/hotstuff"
	"github.com/ethereum/go-ethereum/core/types"
)

// todo: if actually need it
type BlockTree struct {
	tree   *PendingBlockTree
	highQC *hotstuff.QuorumCert // the highest qc
}

func (tr *BlockTree) GetBlockAndCheckHeight(hash common.Hash, height *big.Int) *types.Block {
	parentBlock := tr.GetBlockByHash(hash)
	if parentBlock == nil {
		return nil
	}
	if parentBlock.Number().Cmp(height) != 0 {
		return nil
	}
	return parentBlock
}

func (tr *BlockTree) GetBlockAndCheckRound(hash common.Hash, round *big.Int) *types.Block {
	parentBlock := tr.GetBlockByHash(hash)
	if parentBlock == nil {
		return nil
	}
	_, parentRound, err := extraProposal(parentBlock)
	if err != nil {
		return nil
	}
	if parentRound.Cmp(round) != 0 {
		return nil
	}
	return parentBlock
}

func (tr *BlockTree) GetBlockByHash(hash common.Hash) *types.Block {
	return tr.tree.GetBlockByHash(hash)
}

// Insert insert new block into pending block tree, calculate and return the highestQC
func (tr *BlockTree) Insert(block *types.Block) *hotstuff.QuorumCert {
	return nil
}

func (tr *BlockTree) UpdateHighQC(qc *hotstuff.QuorumCert) {
	if qc == nil || qc.View == nil {
		return
	}
	if tr.highQC == nil || tr.highQC.View == nil {
		tr.highQC = qc
	} else if tr.highQC.View.Round.Cmp(qc.View.Round) < 0 {
		tr.highQC = qc
	}
}

// ProcessCommit commit the block into ledger and pure the `pendingBlockTree`
func (tr *BlockTree) ProcessCommit(hash common.Hash) {

}
