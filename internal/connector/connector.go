package connector

import (
	"errors"
	"fmt"
	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
	"path/filepath"
	"strings"
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
) {
	if sequenceDiagram == nil {
		fmt.Println("TransformSequenceDiagram: sequence diagram is nil")
		return
	}

	fmt.Println("TransformSequenceDiagram: starting transformation of sequence diagram")

	var currentFunction *generator.Method
	var currentFunctionClass *generator.Class
	var lastCalledMethodName string
	//var lastCalledMethodParams []string

	for _, instruction := range sequenceDiagram.Instructions {
		if instruction.Message == nil {
			continue
		}
		message := instruction.Message

		switch message.Type {
		case "->>": // Method call
			// If we don't have a current function yet, this call sets it
			if currentFunction == nil {
				// Right side class must define the method
				targetClass := findClass(classes, message.Right)
				if targetClass == nil {
					fmt.Printf("Couldn't find class: %s\n", message.Right)
					continue
				}
				targetMethod := findMethod(targetClass, message.Name)
				if targetMethod == nil {
					fmt.Printf("Couldn't find method: %s in class %s\n", message.Name, message.Right)
					continue
				}
				currentFunction = targetMethod
				currentFunctionClass = targetClass
				fmt.Printf("Current function set to %s.%s\n", currentFunctionClass.ClassName, currentFunction.Name)
			} else {
				// We are inside a function calling another method
				calledClass := findClass(classes, message.Right)
				if calledClass == nil {
					fmt.Printf("Couldn't find class: %s\n", message.Right)
					continue
				}
				calledMethod := findMethod(calledClass, message.Name)
				if calledMethod == nil {
					fmt.Printf("Couldn't find method: %s in class %s\n", message.Name, message.Right)
					continue
				}

				// Prepare parameters for the call
				callParams := []generator.Attribute{}
				// `message.Parameters` are the actual arguments (like 'user', 'password')
				// `calledMethod.Parameters` contain the defined parameter types and names
				for i, paramArg := range message.Parameters {
					if i < len(calledMethod.Parameters) {
						definedParam := calledMethod.Parameters[i]
						callParams = append(callParams, generator.Attribute{
							Name: paramArg,
							Type: definedParam.Type,
						})
					} else {
						// If mismatch, handle gracefully
						callParams = append(callParams, generator.Attribute{
							Name: paramArg,
							Type: "String", // fallback or handle differently
						})
					}
				}

				body := generator.Body{
					IsVariable:        true,
					FunctionName:      message.Name,
					ObjFuncParameters: callParams,
					Variable: generator.Attribute{
						Name: fmt.Sprintf("temp%s", capitalize(message.Name)),
						Type: calledMethod.ReturnType,
					},
				}
				currentFunction.MethodBody = append(currentFunction.MethodBody, body)
				fmt.Printf("Added variable for method call: %s to method: %s in class: %s\n", message.Name, currentFunction.Name, currentFunctionClass.ClassName)

				lastCalledMethodName = message.Name
				//lastCalledMethodParams = message.Parameters
			}

		case "-->>": // Return value
			if currentFunction == nil {
				fmt.Println("No context for current function. Skipping return value assignment.")
				continue
			}
			if lastCalledMethodName == "" {
				fmt.Printf("No previous method call to assign return value: %s\n", message.Name)
				continue
			}

			// Find the variable from the last call and rename it
			updated := false
			for i, body := range currentFunction.MethodBody {
				if body.IsVariable && body.FunctionName == lastCalledMethodName {
					currentFunction.MethodBody[i].Variable.Name = message.Name
					updated = true
					fmt.Printf("Assigned return value name: %s to variable in method: %s of class: %s\n", message.Name, currentFunction.Name, currentFunctionClass.ClassName)
					break
				}
			}

			if !updated {
				fmt.Printf("Couldn't assign return value name for method: %s\n", currentFunction.Name)
			}

			lastCalledMethodName = ""
			//lastCalledMethodParams = nil
		}
	}

	fmt.Println("TransformSequenceDiagram: completed transformation")
}

func processMessageInstruction(message *reader.Message, classes map[string]*generator.Class) {
	className := message.Right
	methodName := message.Name

	// Find the class
	class := findClass(classes, className)
	if class == nil {
		fmt.Printf("Couldn't find class: %s\n", className)
		return
	}

	// Find the method in the class
	method := findMethod(class, methodName)
	if method == nil {
		fmt.Printf("Couldn't find method: %s in class %s\n", methodName, className)
		return
	}

	// Print success
	fmt.Printf("Found class: %s and method: %s\n", class.ClassName, method.Name)
}

// Helper functions
func findClass(classes map[string]*generator.Class, className string) *generator.Class {
	if class, exists := classes[className]; exists {
		return class
	}
	return nil
}

func findMethod(class *generator.Class, methodName string) *generator.Method {
	for i, method := range class.Methods {
		if method.Name == methodName {
			return &class.Methods[i]
		}
	}
	return nil
}

func capitalize(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(string(str[0])) + str[1:]
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
