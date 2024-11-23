package connector

import (
	"errors"
	"fmt"
	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
	"path/filepath"
)

// TransformClassDiagram transforms a parsed ClassDiagram into JavaClass structs and generates code.
func TransformClassDiagram(
	classDiagram *reader.ClassDiagram,
	classTemplatePath, interfaceTemplatePath, outputDir string,
) error {
	if classDiagram == nil {
		return errors.New("class diagram is nil")
	}

	// Maps for storing generated classes
	classes := make(map[string]*generator.JavaClass)

	// Parse each instruction in the class diagram
	for _, instruction := range classDiagram.Instructions {
		if instruction.Member != nil {
			className := instruction.Member.Class

			// Create JavaClass if not exists
			if _, exists := classes[className]; !exists {
				classes[className] = &generator.JavaClass{
					ClassName:  className,
					Attributes: []generator.Attribute{},
					Methods:    []generator.Method{},
				}
			}

			// Process attributes and methods
			member := instruction.Member
			if member.Attribute != nil {
				classes[className].Attributes = append(classes[className].Attributes, generator.Attribute{
					AccessModifier: parseVisibility(member.Visibility),
					Name:           member.Attribute.Name,
					Type:           member.Attribute.Type,
				})
			} else if member.Operation != nil {
				method := generator.Method{
					AccessModifier: parseVisibility(member.Visibility),
					Name:           member.Operation.Name,
					Type:           member.Operation.Return,
					Parameters:     []generator.Attribute{},
				}
				for _, param := range member.Operation.Parameters {
					method.Parameters = append(method.Parameters, generator.Attribute{
						Name: param.Name,
						Type: param.Type,
					})
				}
				classes[className].Methods = append(classes[className].Methods, method)
			}
		}
	}

	// Handle relationships (inheritance)
	for _, instruction := range classDiagram.Instructions {
		if instruction.Relationship != nil {
			rel := instruction.Relationship
			leftClass := rel.LeftClass
			rightClass := rel.RightClass

			if rel.Type == "--|" || rel.Type == "<|--" { // Inheritance
				if child, exists := classes[leftClass]; exists {
					child.Inherits = rightClass
				}
			}
		}
	}

	// Generate Java classes
	for _, class := range classes {
		outputPath := filepath.Join(outputDir, class.ClassName+".java")
		err := generator.GenerateJavaClass(classTemplatePath, outputPath, *class)
		if err != nil {
			return fmt.Errorf("failed to generate class %s: %v", class.ClassName, err)
		}
	}

	// Note: Currently, no interface handling is implemented.
	// If needed, add interface parsing logic and use generator.GenerateJavaInterface.

	return nil
}

// parseVisibility maps Mermaid visibility symbols to Java access modifiers
func parseVisibility(vis string) string {
	switch vis {
	case "+", "public":
		return "public"
	case "-", "private":
		return "private"
	case "#", "protected":
		return "protected"
	default:
		return "private" // Default to private if unspecified
	}
}
