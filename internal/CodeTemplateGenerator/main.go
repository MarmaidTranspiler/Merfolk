package generator

/*
import (
	"fmt"
	"os"
	"text/template"
)

type JavaClass struct {
	ClassName   string
	Abstraction string
	Inherits    string
	Attributes  []Attribute
	Methods     []Method
}

// Attribute ClassVariable/ True, falls es sich um eine Klassenvariable handelt
// Attribute Constant/ True, falls es sich um eine Konstante handelt
type Attribute struct {
	AccessModifier string
	Name           string
	Type           string
	ClassVariable  bool
	Constant       bool
	Value          any
}

type Method struct {
	AccessModifier string
	Name           string
	Type           string
	Parameters     []Attribute
	Body           string
}

type InterfaceClass struct {
	InterfaceName      string
	Inherits           string
	AbstractAttributes []Attribute
	AbstractMethods    []Method
}

func main() {
	// Print the current working directory for debugging
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fmt.Println("Current Working Directory:", cwd)

	funcMap := template.FuncMap{
		// Funktion für die Standardwerte abhängig vom Typ
		"defaultZero": func(typeName string) string {
			switch typeName {
			case "int":
				return "0"
			case "boolean":
				return "false"
			case "double", "float":
				return "0.0"
			case "String":
				return "\"\""
			default:
				return "null"
			}
		},
		// Funktion zur Großschreibung des ersten Buchstabens
		"title": func(str string) string {
			if len(str) == 0 {
				return ""
			}
			return string(str[0]^32) + str[1:]
		},
		"stringFormation": func(typeName string, value any) any {
			// Fehler bei nil verhindern
			if value == nil {
				return "null"
			}
			switch typeName {
			case "String":
				return fmt.Sprintf("\"%v\"", value)
			default:
				return value
			}
		},
	}

	class_temp, err := template.New("CompleteClassTemplate.tmpl").Funcs(funcMap).ParseFiles("templates/CompleteClassTemplate.tmpl")

	if err != nil {
		panic(err)
	}

	interface_temp, err := template.New("InterfaceTemplate.tmpl").Funcs(funcMap).ParseFiles("templates/InterfaceTemplate.tmpl")

	if err != nil {
		panic(err)
	}

	datenClassDiagram := JavaClass{
		ClassName: "TestClass",
		Inherits:  "Test",
		Attributes: []Attribute{
			{"private", "name", "String", false, false, ""},
			{"public", "age", "int", false, false, ""},
			{"public", "numbPopulation", "int", true, false, "8 Milliarden"},
			{"public", "PI", "double", true, true, 3.124},
			{"public", "MAXAGE", "int", false, true, ""},
		},
	}
	err = class_temp.Execute(os.Stdout, datenClassDiagram)
	if err != nil {
		panic(err)
	}

	datenInterfaceDiagram := InterfaceClass{
		InterfaceName: "InterfaceTest",
		Inherits:      "",
		AbstractAttributes: []Attribute{
			{"", "coll", "Database", false, false, "new Database"},
			{"", "Jahrtausend", "int", false, false, 2100},
			{"", "name", "String", true, false, "Helmut"},
			{"", "PI", "double", true, true, ""},
			{"", "MAXAGE", "int", false, true, ""},
		},
		AbstractMethods: []Method{
			{Type: "void", Name: "doSomething"},
			{Type: "String", Name: "getName"},
		},
	}

	err = interface_temp.Execute(os.Stdout, datenInterfaceDiagram)
	if err != nil {
		panic(err)
	}

}
*/
