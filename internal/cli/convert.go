/*package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
)

func Convert(args []string) {
	if len(args) < 2 {
		fmt.Println("specify both input and output directory")
		return
	}

	inputDir := args[0]
	outputDir := args[1]

	_, err := os.Stat(inputDir)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	_, err = os.Stat(outputDir)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	mdMatches, err := fs.Glob(os.DirFS(inputDir), "*.md")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var files []string
	for _, match := range mdMatches {
		files = append(files, path.Join(inputDir, match))
	}

	var diagrams []reader.Diagram
	for _, file := range files {
		localDiagrams, err := reader.ParseFile(file)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		diagrams = append(diagrams, localDiagrams...)
	}

	// further conversion code here
	// this printing is just supposed to show parsing works
	for _, diagram := range diagrams {
		marshaled, err := json.MarshalIndent(diagram, "", "   ")
		if err != nil {
			panic("oh no")
		}
		fmt.Println(string(marshaled))
	}
}*/

package cli

import (
	"bytes"
	_ "encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
)

func Convert(args []string) {
	if len(args) < 2 {
		fmt.Println("Specify both input and output directory.")
		return
	}

	inputDir := args[0]
	outputDir := args[1]

	// Check and create output directory if needed
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			fmt.Println("Failed to create output directory:", err)
			return
		}
	}

	mdMatches, err := fs.Glob(os.DirFS(inputDir), "*.md")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var files []string
	for _, match := range mdMatches {
		files = append(files, path.Join(inputDir, match))
	}

	var diagrams []reader.Diagram
	for _, file := range files {
		localDiagrams, err := reader.ParseFile(file)
		if err != nil {
			fmt.Println("Error parsing file", file, ":", err)
			continue
		}
		diagrams = append(diagrams, localDiagrams...)
	}

	for _, diagram := range diagrams {
		if diagram.IsClass {
			javaClasses, javaInterfaces := transformClassDiagram(diagram.Class)
			// Generate Java code for classes
			for _, javaClass := range javaClasses {
				err := generateJavaClass(javaClass, outputDir)
				if err != nil {
					fmt.Println("Error generating Java class:", err)
				}
			}
			// Generate Java code for interfaces
			for _, javaInterface := range javaInterfaces {
				err := generateJavaInterface(javaInterface, outputDir)
				if err != nil {
					fmt.Println("Error generating Java interface:", err)
				}
			}
		} else if diagram.IsSequence {
			fmt.Println("Sequence diagrams are not supported yet.")
		} else {
			fmt.Println("Unknown diagram type.")
		}
	}
}

// Template functions
var templateFuncs = template.FuncMap{
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
		return strings.Title(str)
	},
	"stringFormation": func(typeName string, value interface{}) interface{} {
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
	"join": func(items []string, sep string) string {
		return strings.Join(items, sep)
	},
}

// generateJavaClass generates Java code for a class.
func generateJavaClass(javaClass generator.JavaClass, outputDir string) error {
	tmpl, err := template.New("class").Funcs(templateFuncs).
		ParseFiles("CompleteClassTemplate.tmpl")
	if err != nil {
		return err
	}

	var code bytes.Buffer
	err = tmpl.Execute(&code, javaClass)
	if err != nil {
		return err
	}

	filePath := path.Join(outputDir, javaClass.ClassName+".java")
	return os.WriteFile(filePath, code.Bytes(), 0644)
}

// generateJavaInterface generates Java code for an interface.
func generateJavaInterface(
	javaInterface generator.InterfaceClass,
	outputDir string,
) error {
	tmpl, err := template.New("interface").Funcs(templateFuncs).
		ParseFiles("InterfaceTemplate.tmpl")
	if err != nil {
		return err
	}

	var code bytes.Buffer
	err = tmpl.Execute(&code, javaInterface)
	if err != nil {
		return err
	}

	filePath := path.Join(outputDir, javaInterface.InterfaceName+".java")
	return os.WriteFile(filePath, code.Bytes(), 0644)
}
