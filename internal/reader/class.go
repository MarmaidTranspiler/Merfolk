package reader

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type ClassDiagram struct {
	Instructions []*ClassInstruction `Break* @@* Break*`
}

type ClassInstruction struct {
	Relationship *Relationship `  @@`
	Member       *ClassMember  `| @@`
	Annotation   *Annotation   `| @@`
}

type Relationship struct {
	LeftClass        string `@Word`
	LeftCardinality  string `@Cardinality?`
	Type             string `@Relationship`
	RightCardinality string `@Cardinality?`
	RightClass       string `@Word Break?`
	Label            string `( ":" @Word Break)?`
}

type ClassMember struct {
	Class      string     `@Word ":"`
	Visibility string     `@Visibility?`
	Operation  *Operation `@@?`
	Attribute  *Attribute `@@?`
}

type Operation struct {
	Name       string       `@Word`
	Parameters []*Parameter `"(" ( @@ ( "," @@ )* )? ")" Break?`
	Return     string       `(@Word Break)?`
}

type Parameter struct {
	Type string `( @Word (?= Word) )?`
	Name string `@Word`
}

type Attribute struct {
	Type string `( @Word (?= Word) )?`
	Name string `@Word Break`
}

type Annotation struct {
	Name  string `"<<" @Word ">>"`
	Class string `@Word Break`
}

var (
	ClassDiagramLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"diagramType", `classDiagram`},
		{"Word", `[a-zA-Z]\w*`},
		{"Break", `\n`},
		{"Claw", `(<<)|(>>)`},
		{"Special", `[,:\(\)]`},
		{"Relationship", `(<\||\*|o|<)?(--|\.\.)(\|>|\*|o|>)?`},
		{"Visibility", `[+\-#~]`},
		{"Cardinality", `\"(1|(0\.\.1)|\*|(1\.\.\*))\"`},
		{"comment", `%%[^\n]*`},
		{"whitespace", `[ \t\r]+`},
	})

	ClassDiagramParser = participle.MustBuild[ClassDiagram](
		participle.Lexer(ClassDiagramLexer),
		participle.Unquote("Cardinality"),
	)
)
