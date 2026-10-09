package sample

func layout(node *Node) {
	visit(
		func(value *Node) Value { return value.Children[0].(*Branch).Items[0].(*Leaf).Value },
	)

	write(
		byte(0<<6|4<<0|node.Kind<<3), // first
		0<<6|4<<3|5<<0,               // second
	)

	aliases["short"] = first     /* first alias */
	aliases["long-name"] = second /* second alias */

	values := []uint64{
		0x0,  /* zero */
		0x10, /* sixteen */
	}

	_ = node.Kind /* kind */ /* retained */
}
