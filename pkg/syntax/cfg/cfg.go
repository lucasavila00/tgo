// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in ../../../third_party/go/LICENSE.

// Package cfg builds control-flow graphs from TGo syntax ADTs.
package cfg

import "tgo/pkg/syntax"

// CFG is the control-flow graph of one function body.
type CFG struct {
	Blocks []*Block
}

// Block is one basic block.
type Block struct {
	Nodes []syntax.Node
	Succs []*Block
	Index int32
	Live  bool
	Kind  BlockKind
	Stmt  *syntax.Statement

	returns bool
	succs   [2]*Block
}

// BlockKind identifies a block's role.
type BlockKind uint8

// BlockKind values identify control-flow block roles.
const (
	KindInvalid BlockKind = iota
	KindUnreachable
	KindBody
	KindForBody
	KindForDone
	KindForLoop
	KindForPost
	KindIfDone
	KindIfElse
	KindIfThen
	KindLabel
	KindRangeBody
	KindRangeDone
	KindRangeLoop
	KindSelectCaseBody
	KindSelectDone
	KindSelectAfterCase
	KindSwitchCaseBody
	KindSwitchDone
	KindSwitchNextCase
)

// New builds a control-flow graph from one syntax block.
func New(body *syntax.BlockStatement, mayReturn func(*syntax.Expression) bool) *CFG {
	builder := graphBuilder{mayReturn: mayReturn}
	bodyNode := syntax.StatementBlock{Value: body}.Statement()
	builder.current = builder.newBlock(KindBody, &bodyNode)
	builder.statement(&bodyNode)

	queue := make([]*Block, 0, len(builder.blocks))
	queue = append(queue, builder.blocks[0])
	for len(queue) > 0 {
		block := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if !block.Live {
			block.Live = true
			queue = append(queue, block.Succs...)
		}
	}
	return &CFG{Blocks: builder.blocks}
}
