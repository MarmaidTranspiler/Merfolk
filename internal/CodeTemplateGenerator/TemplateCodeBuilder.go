package CodeTemplateGenerator

import (
	"fmt"
	"os"
	"reflect"
	"text/template"
)

// TemplateGeneratorUtility Utility functions for creating templates
func TemplateGeneratorUtility() template.FuncMap {
	return template.FuncMap{
		// Returns the default value for the given data type
		// For primitive types like int, boolean, and double, it returns their default zero value
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
				//Default value for object types
				return "null"
			}
		},
		// Capitalizes the first letter of the input string, leaving the rest unchanged
		"title": func(str string) string {
			// If the string is empty return empty string
			if len(str) == 0 {
				return ""
			}
			return string(str[0]^32) + str[1:]
		},
		// Formats the given string so that the result is enclosed in double quotes ("")
		// If the type is not a String, the value is returned unchanged
		"stringFormation": func(typeName string, value any) any {
			// Return nil if the value was nil
			if value == nil {
				return "null"
			}
			switch typeName {
			// Enclose String values in double quotes using formatted output
			case "String":
				return fmt.Sprintf("\"%v\"", value)
			default:
				// Return the value unchanged for non-String types
				return value
			}
		},
	}
}

// GenerateJavaCode Generate out of the data an Interface or Java class
// The data structure can be of type Class or Interface
// Parameters:
// - dataStruct: The input data structure (either a Class or Interface)
// - outputPath: The directory where the generated file will be saved
// - outputFileName: The name of the generated file (without the ".java" extension)
// - templateFile: The path to the template file used for code generation
func GenerateJavaCode[T Class | Interface](dataStruct T, outputPath string, outputFileName string, templateFile string) error {

	// Load the template file and add utility functions for template processing
	templateStruct, err := template.New(templateFile).Funcs(TemplateGeneratorUtility()).ParseFiles(templateFile)

	// Check if the provided data structure is an Interface
	// If it is, prefix the output file name with "I"
	if reflect.TypeOf(dataStruct).Name() == "Interface" {
		outputFileName = "I" + outputFileName
	}

	// Error handling that may occur while loading the template
	if err != nil {
		panic(err)
	}

	// Create output .java file where the generated code will then be parsed
	file, err := os.Create(outputPath + outputFileName + ".java")
	// Return an error if the file could not be created
	if err != nil {
		return fmt.Errorf("failed to create %v file: %w", file, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	// Pour/Fill the template with data from the provided data structure and write it to the file
	err = templateStruct.Execute(file, dataStruct)
	// Return an Error if there was an error while executing the template
	if err != nil {
		return fmt.Errorf("failed to fill the %s template: %w", templateFile, err)
	}

	// If everything was successful return nil
	return nil
}
