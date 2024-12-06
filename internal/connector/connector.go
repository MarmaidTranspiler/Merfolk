package connector

import (
	"errors"
	"fmt"
	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
	"path/filepath"
)

func TransformClassDiagram(
	classDiagram *reader.ClassDiagram,
	classTemplatePath, interfaceTemplatePath, outputDir string,
) (map[string]*generator.Class, map[string]*generator.Interface, error) {
	if classDiagram == nil {
		fmt.Println("TransformClassDiagram: class diagram is nil")
		return nil, nil, errors.New("class diagram is nil")
	}

	fmt.Println("TransformClassDiagram: starting transformation of class diagram")

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
					fmt.Printf("TransformClassDiagram: created new interface entry for %s\n", className)
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
					fmt.Printf("TransformClassDiagram: created new class entry for %s\n", className)
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
					Value:           fmt.Sprintf("new %s()", member.Attribute.Type), // Set default object creation
				}

				if isInterface {
					interfaces[className].AbstractAttributes = append(interfaces[className].AbstractAttributes, attr)
					classes[className].Attributes = append(classes[className].Attributes, attr)

					fmt.Printf("TransformClassDiagram: added attribute %s to interface %s\n", attr.Name, className)
				} else {
					classes[className].Attributes = append(classes[className].Attributes, attr)
					fmt.Printf("TransformClassDiagram: added attribute %s to class %s\n", attr.Name, className)
				}
			} else if member.Operation != nil {
				method := generator.Method{
					AccessModifier: parseVisibility(member.Visibility),
					Name:           member.Operation.Name,
					IsStatic:       false,
					ReturnType:     member.Operation.Return,
					Parameters:     []generator.Attribute{},
					MethodBody:     []generator.Body{},
					ReturnValue:    fmt.Sprintf("%sResult", member.Operation.Name),
				}

				for _, param := range member.Operation.Parameters {
					method.Parameters = append(method.Parameters, generator.Attribute{
						Name: param.Name,
						Type: param.Type,
					})

					// If the parameter type is a class, ensure it exists as a normal instance attribute
					if _, exists := classes[param.Type]; exists {
						classVar := generator.Attribute{
							AccessModifier:  "private",
							Name:            fmt.Sprintf("%sInstance", param.Type),
							Type:            param.Type,
							IsClassVariable: false,
							IsConstant:      false,
							Value:           fmt.Sprintf("new %s()", param.Type), // Default object creation
						}
						classes[className].Attributes = append(classes[className].Attributes, classVar)
						fmt.Printf("TransformClassDiagram: added class dependency attribute %sInstance to class %s\n", param.Type, className)
					}
				}

				if isInterface {
					interfaces[className].AbstractMethods = append(interfaces[className].AbstractMethods, method)
					fmt.Printf("TransformClassDiagram: added abstract method %s to interface %s\n", method.Name, className)
				} else {
					classes[className].Methods = append(classes[className].Methods, method)
					fmt.Printf("TransformClassDiagram: added method %s to class %s\n", method.Name, className)
				}
			}
		}
	}

	fmt.Println("TransformClassDiagram: generating Java code for interfaces")
	for _, iface := range interfaces {
		fmt.Printf("TransformClassDiagram: generating interface %s\n", iface.InterfaceName)
		err := generator.GenerateJavaCode(*iface, filepath.Clean(outputDir)+"/", iface.InterfaceName, interfaceTemplatePath)
		if err != nil {
			fmt.Printf("TransformClassDiagram: failed to generate interface %s: %v\n", iface.InterfaceName, err)
			return nil, nil, fmt.Errorf("failed to generate interface %s: %w", iface.InterfaceName, err)
		}
	}

	fmt.Println("TransformClassDiagram: transformation completed successfully")
	return classes, interfaces, nil
}

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

			// Retrieve method return type from the class diagram
			methodReturnType := "void" // Default if method is not found
			if class, exists := classes[rightClass]; exists {
				for _, method := range class.Methods {
					if method.Name == message.Name {
						methodReturnType = method.ReturnType
						break
					}
				}
			}

			// Add the method to the sender's class with the correct return type
			method := generator.Method{
				AccessModifier: "public",
				Name:           message.Name,
				ReturnType:     methodReturnType,
				Parameters: []generator.Attribute{
					{Name: "sender", Type: leftClass},
					{Name: "receiver", Type: rightClass},
				},
				MethodBody: []generator.Body{
					{
						IsObjectCreation:  false,
						FunctionName:      fmt.Sprintf("%sInstance.%s", rightClass, message.Name),
						ObjFuncParameters: []generator.Attribute{},
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
