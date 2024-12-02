package connector

import (
	"errors"
	"fmt"
	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
	"path/filepath"
)

// TransformClassDiagram processes a parsed ClassDiagram and adds methods with object generation logic.
func TransformClassDiagram(
	classDiagram *reader.ClassDiagram,
	classTemplatePath, interfaceTemplatePath, outputDir string,
) (map[string]*generator.Class, map[string]*generator.Interface, error) {
	if classDiagram == nil {
		return nil, nil, errors.New("class diagram is nil")
	}

	classes := make(map[string]*generator.Class)
	interfaces := make(map[string]*generator.Interface)

	for _, instruction := range classDiagram.Instructions {
		if instruction.Member != nil {
			className := instruction.Member.Class

			isInterface := instruction.Member.Operation == nil

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
					ReturnType:     member.Operation.Return,
					Parameters:     []generator.Attribute{},
					MethodBody: []generator.Body{
						{
							IsObjectCreation: true,
							ObjectName:       fmt.Sprintf("%sResult", member.Operation.Name),
							ObjectType:       member.Operation.Return,
							ObjFuncParameters: []generator.Attribute{
								{Name: "args", Type: "Object"}, // Placeholder for arguments
							},
						},
					},
					ReturnValue: fmt.Sprintf("%sResult", member.Operation.Name),
				}

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

	// Generate Java code for all interfaces
	for _, iface := range interfaces {
		err := generator.GenerateJavaCode(*iface, filepath.Clean(outputDir)+"/", iface.InterfaceName, interfaceTemplatePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate interface %s: %w", iface.InterfaceName, err)
		}
	}

	return classes, interfaces, nil
}

// TransformSequenceDiagram integrates sequence diagrams into the classes generated from the class diagram.
func TransformSequenceDiagram(
	sequenceDiagram *reader.SequenceDiagram,
	classes map[string]*generator.Class,
	classTemplatePath, outputDir string,
) error {
	if sequenceDiagram == nil {
		return fmt.Errorf("sequence diagram is nil")
	}

	participants := make(map[string]string)

	for _, instruction := range sequenceDiagram.Instructions {
		switch {
		case instruction.Member != nil:
			member := instruction.Member
			participants[member.Name] = member.Name

			if _, exists := classes[member.Name]; exists {
				continue
			}

			classes[member.Name] = &generator.Class{
				ClassName:  member.Name,
				Attributes: []generator.Attribute{},
				Methods:    []generator.Method{},
			}

		case instruction.Message != nil:
			message := instruction.Message
			leftClass := participants[message.Left]
			rightClass := participants[message.Right]

			// Ensure participants have corresponding classes
			if _, exists := classes[leftClass]; !exists {
				classes[leftClass] = &generator.Class{
					ClassName:  leftClass,
					Attributes: []generator.Attribute{},
					Methods:    []generator.Method{},
				}
			}
			if _, exists := classes[rightClass]; !exists {
				classes[rightClass] = &generator.Class{
					ClassName:  rightClass,
					Attributes: []generator.Attribute{},
					Methods:    []generator.Method{},
				}
			}

			// Add the method to the sender's class
			method := generator.Method{
				AccessModifier: "public",
				Name:           message.Name,
				ReturnType:     rightClass,
				Parameters: []generator.Attribute{
					{Name: "sender", Type: leftClass},
					{Name: "receiver", Type: rightClass},
				},
				MethodBody: []generator.Body{
					{
						IsObjectCreation: true,
						ObjectName:       fmt.Sprintf("%sResult", message.Name),
						ObjectType:       rightClass,
					},
				},
				ReturnValue: fmt.Sprintf("%sResult", message.Name),
			}

			if class, exists := classes[leftClass]; exists {
				class.Methods = append(class.Methods, method)
			}
		}
	}

	// Generate Java code for all classes
	for _, class := range classes {
		err := generator.GenerateJavaCode(*class, filepath.Clean(outputDir)+"/", class.ClassName, classTemplatePath)
		if err != nil {
			return fmt.Errorf("failed to generate class %s: %w", class.ClassName, err)
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
		return "private"
	}
}
