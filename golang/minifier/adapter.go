package minifier

import "github.com/antchfx/xmlquery"

type XMLDoc struct {
	root *xmlquery.Node
}

func NewXMLDoc(root *xmlquery.Node) *XMLDoc {
	return &XMLDoc{
		root: root,
	}
}

func (d *XMLDoc) Find(xpath string) []Node {
	var nodes []Node
	elems := xmlquery.Find(d.root, xpath)

	for _, elem := range elems {
		node := NewXMLNode(elem)
		nodes = append(nodes, node)
	}
	return nodes
}

func (d *XMLDoc) Root() *XMLNode {
	return NewXMLNode(d.root)
}

type XMLNode struct {
	node *xmlquery.Node
}

func NewXMLNode(node *xmlquery.Node) *XMLNode {
	return &XMLNode{
		node: node,
	}
}

func (n XMLNode) TagName() string {
	return n.node.Data
}

func (n XMLNode) IsElement() bool {
	return n.node.Type == xmlquery.ElementNode
}

func (n *XMLNode) HasParent() bool {
	return n.node.Parent != nil
}

func (n *XMLNode) GetAttribute(key string) string {
	for _, attr := range n.node.Attr {
		if attr.Name.Local == key {
			return attr.Value
		}
	}
	return ""
}

func (n *XMLNode) GetParent() Node {
	return NewXMLNode(n.node.Parent)
}

func (n *XMLNode) ChildNodes() []Node {
	var nodes []Node

	for c := n.node.FirstChild; c != nil; c = c.NextSibling {
		xn := NewXMLNode(c)
		nodes = append(nodes, xn)
	}
	return nodes
}

func (n *XMLNode) Equal(n1 Node) bool {
	xn1, ok := n1.(*XMLNode)
	if !ok {
		return false
	}

	return n.node == xn1.node
}

func (n *XMLNode) Index() int {
	if n.node.Parent == nil {
		return 1
	}

	idx := 0
	parent := n.node.Parent
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xmlquery.ElementNode && c.Data == n.node.Data {
			idx += 1
			if c == n.node {
				return idx
			}

		}
	}
	return 1
}
