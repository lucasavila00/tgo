package main

import "fmt"

type Tree struct {
	tgoTag      TreeTag
	tgoVariant0 TreeLeaf
	tgoVariant1 TreeBranch
}
type TreeTag uint8

const (
	TreeTagLeaf   TreeTag = 0
	TreeTagBranch TreeTag = 1
)

func (tgo15 Tree) Tag() TreeTag {
	return tgo15.tgoTag
}

type TreeLeaf struct{ Value int }

func NewTreeLeaf(tgo16 int) Tree {
	return Tree{tgoTag: TreeTagLeaf, tgoVariant0: TreeLeaf{tgo16}}
}
func (tgo17 Tree) LeafPayload() TreeLeaf {
	return tgo17.tgoVariant0
}

type TreeBranch struct {
	Children []Tree
	Parent   *Tree
}

func NewTreeBranch(tgo18 []Tree, tgo19 *Tree) Tree {
	return Tree{tgoTag: TreeTagBranch, tgoVariant1: TreeBranch{tgo18, tgo19}}
}
func (tgo20 Tree) BranchPayload() TreeBranch {
	return tgo20.tgoVariant1
}

func total(tree Tree) int {
	var tgo1 int
	{
		var value_tgo2 Tree = tree
		switch value_tgo2.Tag() {
		case TreeTagLeaf:
			var value_tgo3 TreeLeaf = value_tgo2.LeafPayload()
			tgo1 = value_tgo3.Value
		default:
			var value_tgo4 TreeBranch = value_tgo2.BranchPayload()
			sum := 0
			for _, child := range value_tgo4.Children {
				sum += total(child)
			}
			tgo1 = sum
		}
	}
	return tgo1

}
func main() {
	var payload_tgo5 TreeLeaf = TreeLeaf{Value: 3}
	var enum_tgo6 Tree = NewTreeLeaf(payload_tgo5.Value)
	leaf := enum_tgo6
	var value_tgo9 Tree = leaf
	var payload_tgo7 TreeLeaf = TreeLeaf{Value: 4}
	var enum_tgo8 Tree = NewTreeLeaf(payload_tgo7.Value)
	var payload_tgo10 TreeBranch = TreeBranch{Children: []Tree{value_tgo9, enum_tgo8}, Parent: &leaf}
	var enum_tgo11 Tree = NewTreeBranch(payload_tgo10.Children, payload_tgo10.Parent)
	tree := enum_tgo11
	fmt.Println(total(tree), tree.BranchPayload().Parent.LeafPayload().Value)
	var value_tgo12 *Tree = tree.BranchPayload().Parent
	var payload_tgo13 TreeLeaf = TreeLeaf{Value: 9}
	var enum_tgo14 Tree = NewTreeLeaf(payload_tgo13.Value)
	*value_tgo12 = enum_tgo14
	fmt.Println(total(tree), leaf.LeafPayload().Value)
}
