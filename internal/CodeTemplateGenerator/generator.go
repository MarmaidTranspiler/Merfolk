package CodeTemplateGenerator

/*package generator

import (
	"fmt"
	"os"
	"text/template"
)

// JavaClass, Attribute, Method, InterfaceClass remain the same...

type JavaClass struct {
	ClassName   string
	Abstraction string
	Inherits    string
	Attributes  []Attribute
	Methods     []Method
}

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

// GenerateJavaClass generates Java code for a class and writes it to the specified output file
func GenerateJavaClass(templateFile string, outputFile string, classData JavaClass) error {
	funcMap := createFuncMap()
	tmpl, err := template.New("CompleteClassTemplate.tmpl").Funcs(funcMap).ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	err = tmpl.Execute(file, classData)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// GenerateJavaInterface generates Java code for an interface and writes it to the specified output file
func GenerateJavaInterface(templateFile string, outputFile string, interfaceData InterfaceClass) error {
	funcMap := createFuncMap()
	tmpl, err := template.New("InterfaceTemplate.tmpl").Funcs(funcMap).ParseFiles(templateFile)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	err = tmpl.Execute(file, interfaceData)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// Helper function to create the function map for templates
func createFuncMap() template.FuncMap {
	return template.FuncMap{
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
		"title": func(str string) string {
			if len(str) == 0 {
				return ""
			}
			return string(str[0]^32) + str[1:]
		},
		"stringFormation": func(typeName string, value any) any {
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
}
*/
