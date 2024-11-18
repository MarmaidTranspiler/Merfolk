package parser

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type ClassDiagram struct {
	Instructions []*Instruction `@@*`
}

type Instruction struct {
	Relationship *Relationship `  @@`
	Member       *Member       `| @@`
	Annotation   *Annotation   `| @@`
}

type Relationship struct {
	LeftClass        string `@Word`
	LeftCardinality  string `@Cardinality?`
	Type             string `@Relationship`
	RightCardinality string `@Cardinality?`
	RightClass       string `@Word`
	Label            string `( ":" @Word )?`
}

type Member struct {
	Class      string     `@Word ":"`
	Visibility string     `@Visibility?`
	Attribute  *Parameter `@@?`
	Operation  *Operation `@@?`
}

type Operation struct {
	Name       string       `@Word`
	Parameters []*Parameter `"(" ( @@ ( ","? @@ )* )? ")"`
	Return     string       `@Word?`
}

type Parameter struct {
	Type string `( @Word (?= Word) )?`
	Name string `@Word`
}

type Annotation struct {
	Name  string `"<<" @Word ">>"`
	Class string `@Word`
}

var (
	Lexer = lexer.MustSimple([]lexer.SimpleRule{
		{"Word", `[a-zA-Z]\w*`},
		{"Claw", `(<<)|(>>)`},
		{"Special", `[,:\(\)]`},
		{"Relationship", `(<\||\*|o|<)?(--|\.\.)(\|>|\*|o|>)?`},
		{"Visibility", `[+\-#~]`},
		{"Cardinality", `\"(1|(0\.\.1)|\*|(1\.\.\*))\"`},
		{"comment", `%%[^\n]*`},
		{"whitespace", `\s+`},
	})

	Parser = participle.MustBuild[ClassDiagram](
		participle.Lexer(Lexer),
		participle.Unquote("Cardinality"),
	)
)
