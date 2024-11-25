package connector

import (
	"errors"
	"fmt"

	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
)

// TransformClassDiagram processes a parsed ClassDiagram, converts it into Class and Interface structs,
// and uses the generator package to produce Java code.
func TransformClassDiagram(
	classDiagram *reader.ClassDiagram,
	classTemplatePath, interfaceTemplatePath, outputDir string,
) error {
	if classDiagram == nil {
		return errors.New("class diagram is nil")
	}

	// Maps for storing classes and interfaces
	classes := make(map[string]*generator.Class)
	interfaces := make(map[string]*generator.Interface)

	// Parse each instruction in the class diagram
	for _, instruction := range classDiagram.Instructions {
		if instruction.Member != nil {
			className := instruction.Member.Class

			// Determine whether this is a class or an interface
			isInterface := instruction.Member.Operation == nil

			// Initialize Class or Interface structs
			if isInterface {
				if _, exists := interfaces[className]; !exists {
					interfaces[className] = &generator.Interface{
						InterfaceName:      className,
						Inherits:           []string{},
						AbstractAttributes: []generator.Attribute{},
						AbstractMethods:    []generator.Method{},
					}
				}
			} else {
				if _, exists := classes[className]; !exists {
					classes[className] = &generator.Class{
						ClassName:   className,
						Abstraction: []string{},
						Inherits:    "",
						Attributes:  []generator.Attribute{},
						Methods:     []generator.Method{},
					}
				}
			}

			// Process attributes and methods
			member := instruction.Member
			if member.Attribute != nil {
				attr := generator.Attribute{
					AccessModifier:  parseVisibility(member.Visibility),
					Name:            member.Attribute.Name,
					Type:            member.Attribute.Type,
					IsClassVariable: false,
					IsConstant:      false,
					Value:           nil,
				}

				if isInterface {
					interfaces[className].AbstractAttributes = append(interfaces[className].AbstractAttributes, attr)
				} else {
					classes[className].Attributes = append(classes[className].Attributes, attr)
				}
			} else if member.Operation != nil {
				method := generator.Method{
					AccessModifier: parseVisibility(member.Visibility),
					Name:           member.Operation.Name,
					IsStatic:       false,
					ReturnType:     member.Operation.Return, // Updated field name
					Parameters:     []generator.Attribute{},
					MethodBody:     []generator.Body{}, // Updated field name
					ReturnValue:    "",                 // Optional: Initialize to default
				}

				// Add parameters to the method
				for _, param := range member.Operation.Parameters {
					method.Parameters = append(method.Parameters, generator.Attribute{
						Name: param.Name,
						Type: param.Type,
					})
				}

				if isInterface {
					interfaces[className].AbstractMethods = append(interfaces[className].AbstractMethods, method)
				} else {
					classes[className].Methods = append(classes[className].Methods, method)
				}
			}
		}
	}

	// Generate Java classes
	for _, class := range classes {
		err := generator.GenerateJavaCode(*class, outputDir+"/", class.ClassName, classTemplatePath)
		if err != nil {
			return fmt.Errorf("failed to generate class %s: %w", class.ClassName, err)
		}
	}

	// Generate Java interfaces
	for _, iface := range interfaces {
		err := generator.GenerateJavaCode(*iface, outputDir+"/", iface.InterfaceName, interfaceTemplatePath)
		if err != nil {
			return fmt.Errorf("failed to generate interface %s: %w", iface.InterfaceName, err)
		}
	}

	return nil
}

// parseVisibility maps Mermaid visibility symbols to Java access modifiers.
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
