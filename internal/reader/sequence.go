package reader

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type SequenceDiagram struct {
	Instructions []*SequenceInstruction `Break* @@* Break*`
}

type SequenceInstruction struct {
	Message *Message        `  @@`
	Loop    *Loop           `| @@`
	Alt     *Alt            `| @@`
	End     *End            `| @@`
	Member  *SequenceMember `| @@`
	Life    *Life           `| @@`
	Switch  *Switch         `| @@`
}

type Message struct {
	Left        string   `@Word`
	Type        string   `@Arrow`
	Right       string   `@Word ":"`
	Name        string   `@Word Break*`
	DefaultCall bool     `@( "(" ")" )? Break*`
	Parameters  []string `( "(" (@Word | @String) ( "," (@Word | @String) )* ")" )? Break*`
}

type Loop struct {
	Definition []string `"loop" @Word+ Break+`
}

type Alt struct {
	Definition []string `"alt" @Word+ Break+`
}

type End struct {
	Clear string `@"end" Break+`
}

type SequenceMember struct {
	Type string `@( "participant" | "actor" )`
	Name string `@Word ("as" Word)? Break+`
}

type Life struct {
	Type string `@( "create" | "destroy" )`
	On   string `@("participant" | "actor")?`
	Name string `@Word Break+`
}

type Switch struct {
	Type string `@( "activate" | "deactivate" )`
	Name string `@Word Break+`
}

var (
	SequenceDiagramLexer = lexer.MustSimple([]lexer.SimpleRule{
		{"diagramType", `sequenceDiagram`},
		{"Keyword", `(?i)(loop|alt|end|participant|actor|as|create|destroy|(de)?activate)`},
		{"Special", `[:,\(\)]`},
		{"Break", `\n`},
		{"Arrow", `((<<)?--?>>)|(--?[>x)])`},
		{"Word", `[a-zA-Z]\w*`},
		{"comment", `%%[^\n]*`},
		{"note", `(?i)note[^\n]*`},
		{"whitespace", `\s+`},
		{"String", `"(?:[^"\\]|\\.)*"`}, // Handles double-quoted strings
		{"Null", `<null>`},              // Add support for <null>

	})

	SequenceDiagramParser = participle.MustBuild[SequenceDiagram](
		participle.Lexer(SequenceDiagramLexer),
	)
)
