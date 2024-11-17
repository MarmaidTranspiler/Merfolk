package main

import (
	"os"
	"text/template"
)

type JavaClass struct {
	ClassName  string
	Inherits   string
	Attributes []Attribute
	Methods    []Method
}

// Attribute ClassVariable/ True, falls es sich um eine Klassenvariable handelt */
// Attribute Constant/ True, falls es sich um eine Konstante handelt
type Attribute struct {
	AccessModifier string
	Name           string
	Type           string
	ClassVariable  bool
	Constant       bool
}

type Method struct {
	AccessModifier string
	Name           string
	Type           string
	Parameters     []Attribute
	Body           string
}

func main() {

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
	}

	template_class, err := template.New("CompleteClassTemplate.tmpl").Funcs(funcMap).ParseFiles("CompleteClassTemplate.tmpl")

	if err != nil {
		panic(err)
	}

	daten_class_diagram := JavaClass{
		ClassName: "TestClass",
		Inherits:  "Test",
		Attributes: []Attribute{
			{"private", "name", "string", false, false},
			{"public", "age", "int", false, false},
			{"public", "numbPopulation", "int", true, false},
			{"public", "PI", "double", true, true},
			{"public", "MAXAGE", "int", false, true},
		},
	}
	err = template_class.Execute(os.Stdout, daten_class_diagram)
	if err != nil {
		panic(err)
	}
}
