package reader

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type SequenceDiagram struct {
	Instructions []*SequenceInstruction `@@*`
}

type SequenceInstruction struct {
	Message *Message        `  @@`
	Member  *SequenceMember `| @@`
	Life    *Life           `| @@`
	Switch  *Switch         `| @@`
}

type Message struct {
	Left        string   `@Word`
	Type        string   `@Arrow`
	Right       string   `@Word ":"`
	Name        string   `@Word`
	DefaultCall bool     `@( "(" ")" )?`
	Parameters  []string `( "(" @Word ( "," @Word )* ")" )?`
}

type SequenceMember struct {
	Type string `@( "participant" | "actor" )`
	Name string `@Word ("as" Word)?`
}

type Life struct {
	Type string `@( "create" | "destroy" )`
	On   string `@("participant" | "actor")?`
	Name string `@Word`
}

type Switch struct {
	Type string `@( "activate" | "deactivate" )`
	Name string `@Word`
}

var (
	SequenceDiagramLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"diagramType", `sequenceDiagram`},
		{"Keyword", `(?i)(participant|actor|as|create|destroy|(de)?activate)`},
		{"Special", `[:,\(\)]`},
		{"Arrow", `((<<)?--?>>)|(--?[>x)])`},
		{"Word", `[a-zA-Z]\w*`},
		{"comment", `%%[^\n]*`},
		{"note", `(?i)note[^\n]*`},
		{"whitespace", `\s+`},
	})

	SequenceDiagramReader = participle.MustBuild[SequenceDiagram](
		participle.Lexer(SequenceDiagramLexer),
	)
)
