package main

import (
	"fmt"
	"os"
	"text/template"
)

// Utility functions for creating templates
func templateGeneratorUtility() template.FuncMap {
	return template.FuncMap{
		// Returns the default value for the given data type.
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
		// Capitalizes the first letter of the input string, leaving the rest unchanged
		"title": func(str string) string {
			if len(str) == 0 {
				return ""
			}
			return string(str[0]^32) + str[1:]
		},
		// Formats the given string so that the result is enclosed in double quotes ("")
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

func generateJavaClass(classData JavaClass, outputPath string, outputFileName string) error {

	classTemplate, err := template.New("CompleteClassTemplate.tmpl").Funcs(templateGeneratorUtility()).ParseFiles("CompleteClassTemplate.tmpl")

	if err != nil {
		panic(err)
	}

	file, err := os.Create(outputPath + outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create %s file: %w", file, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	err = classTemplate.Execute(file, classData)
	if err != nil {
		return fmt.Errorf("failed to fill the Java class template: %w", err)
	}

	return nil
}

func generateJavaInterface(interfaceData InterfaceClass, outputPath string, outputFileName string) error {

	interfaceTemplate, err := template.New("InterfaceTemplate.tmpl").Funcs(templateGeneratorUtility()).ParseFiles("InterfaceTemplate.tmpl")
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(outputPath + outputFileName)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	err = interfaceTemplate.Execute(file, interfaceData)
	if err != nil {
		return fmt.Errorf("failed to fill the Java class template: %w", err)
	}

	return nil
}

func test() {

}
