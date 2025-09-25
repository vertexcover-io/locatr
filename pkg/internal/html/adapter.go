package html

import (
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

type Document interface {
	Find(xpath string) []Node
}

type Node interface {
	TagName() string
	IsElement() bool
	HasParent() bool
	GetAttribute(key string) string
	GetParent() Node
	ChildNodes() []Node
	Index() int
	Equal(Node) bool
}

type HTMLDoc struct {
	root *html.Node
}

func NewHTMLDoc(root *html.Node) *HTMLDoc {
	return &HTMLDoc{root: root}
}

func (d *HTMLDoc) Find(xpath string) []Node {
	var nodes []Node
	elems := htmlquery.Find(d.root, xpath)

	for _, elem := range elems {
		node := NewHTMLNode(elem)
		nodes = append(nodes, node)
	}
	return nodes
}

func (d *HTMLDoc) Root() *HTMLNode {
	return NewHTMLNode(d.root)
}

type HTMLNode struct {
	node *html.Node
}

func NewHTMLNode(node *html.Node) *HTMLNode {
	return &HTMLNode{node: node}
}

func (n HTMLNode) TagName() string {
	return n.node.Data
}

func (n HTMLNode) IsElement() bool {
	return n.node.Type == html.ElementNode
}

func (n *HTMLNode) HasParent() bool {
	return n.node.Parent != nil
}

func (n *HTMLNode) GetAttribute(key string) string {
	for _, attr := range n.node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func (n *HTMLNode) GetParent() Node {
	return NewHTMLNode(n.node.Parent)
}

func (n *HTMLNode) ChildNodes() []Node {
	var nodes []Node

	for c := n.node.FirstChild; c != nil; c = c.NextSibling {
		xn := NewHTMLNode(c)
		nodes = append(nodes, xn)
	}
	return nodes
}

func (n *HTMLNode) Equal(n1 Node) bool {
	xn1, ok := n1.(*HTMLNode)
	if !ok {
		return false
	}

	return n.node == xn1.node
}

func (n *HTMLNode) Index() int {
	if n.node.Parent == nil {
		return 1
	}

	idx := 0
	parent := n.node.Parent
	for c := parent.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == n.node.Data {
			idx += 1
			if c == n.node {
				return idx
			}

		}
	}
	return 1
}

func IsValidXPath(xpath, dom string) (bool, error) {
	doc, err := htmlquery.Parse(strings.NewReader(dom))
	if err != nil {
		return false, err
	}

	elem, err := htmlquery.Query(doc, xpath)
	if err != nil {
		return false, err
	}
	return elem != nil, nil
}
