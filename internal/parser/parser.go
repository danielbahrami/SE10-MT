package parser

import "github.com/antlr4-go/antlr/v4"

// Returns the ANTLR parse-tree for an input Cypher string
func Parse(input string) antlr.ParseTree {
	is := antlr.NewInputStream(input)
	lex := NewCypherLexer(is)
	tokens := antlr.NewCommonTokenStream(lex, 0)
	p := NewCypherParser(tokens)
	p.BuildParseTrees = true
	return p.OC_Cypher() // Entry rule from the grammar
}
