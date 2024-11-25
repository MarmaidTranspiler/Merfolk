package CodeTemplateGenerator

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"text/template"
)

// TemplateGeneratorUtility Utility functions for creating templates
func TemplateGeneratorUtility() template.FuncMap {
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

// GenerateJavaCode Generate out of the data an Interface or Java class
func GenerateJavaCode[T Class | Interface](dataStruct T, outputPath string, outputFileName string, templateFile string) error {

	templateStruct, err := //template.New(templateFile).Funcs(TemplateGeneratorUtility()).ParseFiles(templateFile)
		template.New(filepath.Base(templateFile)).Funcs(TemplateGeneratorUtility()).ParseFiles(templateFile)
	if reflect.TypeOf(dataStruct).Name() == "Interface" {
		outputFileName = "I" + outputFileName
	}

	if err != nil {
		panic(err)
	}

	file, err := os.Create(outputPath + outputFileName + ".java")
	if err != nil {
		return fmt.Errorf("failed to create %s file: %w", file, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	err = templateStruct.Execute(file, dataStruct)
	if err != nil {
		return fmt.Errorf("failed to fill the %s template: %w", templateFile, err)
	}

	return nil
}
