package docs

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// wrapPaths puts each reading path of the landing inside a pathCard, so the page can lay the
// paths out as cards without new Markdown syntax.
func wrapPaths(root ast.Node) {
	for _, s := range pathSections(root) {
		card := &pathCard{}
		root.InsertBefore(root, s.blocks[0], card)
		for _, n := range s.blocks {
			card.AppendChild(card, n)
		}
	}
}

var kindPathCard = ast.NewNodeKind("PathCard")

type pathCard struct{ ast.BaseBlock }

func (n *pathCard) Kind() ast.NodeKind { return kindPathCard }

func (n *pathCard) Dump(src []byte, level int) { ast.DumpHelper(n, src, level, nil, nil) }

type pathCardRenderer struct{}

func (pathCardRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindPathCard, func(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
		tag := "</div>\n"
		if entering {
			tag = "<div class=\"path\">\n"
		}
		_, err := w.WriteString(tag)
		return ast.WalkContinue, err
	})
}
