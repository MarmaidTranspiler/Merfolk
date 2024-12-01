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

func TransformSequenceDiagram(
	sequenceDiagram *reader.SequenceDiagram,
	classTemplatePath, outputDir string,
) error {
	if sequenceDiagram == nil {
		return fmt.Errorf("sequence diagram is nil")
	}

	// Initialize class data
	classData := generator.Class{
		Attributes: []generator.Attribute{},
		Methods:    []generator.Method{},
	}

	// Variable to hold the class name
	className := ""

	// Map each instruction in the sequence diagram
	for i, instruction := range sequenceDiagram.Instructions {
		fmt.Printf("Instruction #%d: %+v\n", i, instruction)

		switch {
		case instruction.Member != nil:
			member := instruction.Member
			fmt.Printf("Parsed SequenceMember: Type=%s, Name=%s\n", member.Type, member.Name)

			// Set the class name to the name of the first actor
			if member.Type == "actor" && className == "" {
				className = member.Name
				fmt.Printf("Class name set to actor: %s\n", className)
			}

			// Add participants as attributes (type is the same as their name)
			classData.Attributes = append(classData.Attributes, generator.Attribute{
				AccessModifier: "private",
				Name:           member.Name,
				Type:           member.Name, // Type is the participant's name
			})

		case instruction.Message != nil:
			message := instruction.Message
			fmt.Printf("Parsed Message: Left=%s, Right=%s, Name=%s\n",
				message.Left, message.Right, message.Name)

			// Generate the variable name for this message
			variableName := fmt.Sprintf("%sTo%sMessage", message.Left, message.Right)

			// Construct the method body for variable creation
			methodBody := []generator.Body{
				{
					IsObjectCreation:  true,
					ObjectName:        variableName,
					ObjectType:        "String",
					ObjFuncParameters: []generator.Attribute{},
				},
			}

			// Create a method for the message
			method := generator.Method{
				AccessModifier: "public",
				Name:           message.Name,
				ReturnType:     "String",
				Parameters: []generator.Attribute{
					{Name: "sender", Type: message.Left},
					{Name: "receiver", Type: message.Right},
				},
				MethodBody:  methodBody,
				ReturnValue: variableName, // Return the created variable
			}

			// Add parameters from the message
			for _, param := range message.Parameters {
				method.Parameters = append(method.Parameters, generator.Attribute{
					Name: param,
					Type: "String",
				})
			}

			classData.Methods = append(classData.Methods, method)

		case instruction.Loop != nil:
			loop := instruction.Loop
			fmt.Printf("Loop Instruction Found: IsStart=%t, IsEnd=%t, Definition=%+v\n", loop.IsStart, loop.IsEnd, loop.Definition)

			if loop.IsStart {
				method := generator.Method{
					AccessModifier: "public",
					Name:           "startLoop",
					ReturnType:     "void",
					Parameters:     []generator.Attribute{},
					MethodBody:     []generator.Body{},
				}
				classData.Methods = append(classData.Methods, method)
			}

			if loop.IsEnd {
				method := generator.Method{
					AccessModifier: "public",
					Name:           "endLoop",
					ReturnType:     "void",
					Parameters:     []generator.Attribute{},
					MethodBody:     []generator.Body{},
				}
				classData.Methods = append(classData.Methods, method)
			}

		default:
			fmt.Printf("Unrecognized Instruction: %+v\n", instruction)
		}
	}

	// Ensure class name is set
	if className == "" {
		fmt.Println("No actor found. Ensure the sequence diagram defines at least one actor.")
		fmt.Println("Logging all instructions:")
		for _, instruction := range sequenceDiagram.Instructions {
			fmt.Printf("Instruction: %+v\n", instruction)
		}
		return fmt.Errorf("no actor defined in the sequence diagram to determine class name")
	}
	classData.ClassName = className

	// Generate Java code using the class template
	err := generator.GenerateJavaCode(classData, outputDir+"/", classData.ClassName, classTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to generate Java code for sequence diagram: %w", err)
	}

	fmt.Printf("Successfully generated Java code for sequence diagram with class name: %s\n", className)
	return nil
}
