package minifier

import (
	"strconv"

	"github.com/antchfx/xmlquery"
)

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
		node := XMLNode(*elem)
		nodes = append(nodes, &node)
	}
	return nodes
}

func (d *XMLDoc) IndexOf(node Node) int {
	return node.Index()
}

func (d *XMLDoc) Root() Node {
	n := XMLNode(*d.root)
	return &n
}

type XMLNode xmlquery.Node

func (n XMLNode) TagName() string {
	return n.Data
}

func (n XMLNode) IsElement() bool {
	return n.Type == xmlquery.ElementNode
}

func (n *XMLNode) HasParent() bool {
	return n.Parent != nil
}

func (n *XMLNode) GetAttribute(key string) string {
	for _, attr := range n.Attr {
		if attr.Name.Local == key {
			return attr.Value
		}
	}
	return ""
}

func (n *XMLNode) GetParent() Node {
	cn := XMLNode(*n.Parent)
	return &cn
}

func (n *XMLNode) ChildNodes() []Node {
	var nodes []Node

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		xn := XMLNode(*c)
		nodes = append(nodes, &xn)
	}
	return nodes
}

func (n *XMLNode) Index() int {
	idx, _ := strconv.Atoi(n.GetAttribute("index"))
	return idx
}

// func printPath(doc XMLDoc, node XMLNode) {
// 	xpath := GetOptimalXPath(&doc, &node)
// 	if strings.TrimSpace(xpath) != "" {
// 		fmt.Printf("%s: %s\n", node.Data, xpath)
// 	}

// 	for _, c := range node.ChildNodes() {
// 		n := c.(*XMLNode)
// 		printPath(doc, *n)
// 	}
// }

// func main() {
// 	xmlFile, err := os.Open(fileName)
// 	if err != nil {
// 		log.Fatalf("failed to read file %s, %v", fileName, err)
// 	}
// 	defer xmlFile.Close()

// 	root, err := xmlquery.Parse(xmlFile)
// 	if err != nil {
// 		log.Fatalf("failed to parse xmlfile, %v", err)
// 	}
// 	doc := NewXMLDoc(root)

// 	printPath(*doc, doc.Root())
