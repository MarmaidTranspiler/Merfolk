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
) error {
	if sequenceDiagram == nil {
		fmt.Println("TransformSequenceDiagram: sequence diagram is nil")
		return errors.New("sequence diagram is nil")
	}

	fmt.Println("TransformSequenceDiagram: starting transformation of sequence diagram")

	type methodContext struct {
		class  *generator.Class
		method *generator.Method
	}
	var callStack []methodContext

	// callReturnStack holds references to the caller and line index for each method call
	// Each entry: (callerClass, callerMethod, callLineIndex)
	type callReturnInfo struct {
		callerClass  *generator.Class
		callerMethod *generator.Method
		lineIndex    int
	}
	var callReturnStack []callReturnInfo

	participants := make(map[string]bool)

	var lastObjectIsConstructor bool
	var lastObjectTempVarName string

	getCurrentContext := func() (*generator.Class, *generator.Method) {
		if len(callStack) == 0 {
			return nil, nil
		}
		top := callStack[len(callStack)-1]
		return top.class, top.method
	}

	pushContext := func(c *generator.Class, m *generator.Method) {
		callStack = append(callStack, methodContext{class: c, method: m})
	}

	popContext := func() {
		if len(callStack) > 0 {
			callStack = callStack[:len(callStack)-1]
		}
	}

	findOrCreateDummyClass := func(name string) *generator.Class {
		if classes[name] == nil {
			classes[name] = &generator.Class{
				ClassName:  name,
				Attributes: []generator.Attribute{},
				Methods:    []generator.Method{},
			}
		}
		return classes[name]
	}

	ensureObjectReference := func(
		currentFunction *generator.Method,
		currentFunctionClass *generator.Class,
		targetClassName string,
	) string {
		if currentFunctionClass == nil || currentFunction == nil {
			return strings.ToLower(string(targetClassName[0])) + targetClassName[1:]
		}

		for i, attr := range currentFunctionClass.Attributes {
			if attr.Type == targetClassName {
				return currentFunctionClass.Attributes[i].Name
			}
		}

		for _, bodyLine := range currentFunction.MethodBody {
			if bodyLine.IsObjectCreation && bodyLine.ObjectType == targetClassName {
				return bodyLine.ObjectName
			}
		}

		localVarName := strings.ToLower(string(targetClassName[0])) + targetClassName[1:]
		newBodyLine := generator.Body{
			IsObjectCreation: true,
			ObjectName:       localVarName,
			ObjectType:       targetClassName,
		}
		currentFunction.MethodBody = append([]generator.Body{newBodyLine}, currentFunction.MethodBody...)
		return localVarName
	}

	for _, instruction := range sequenceDiagram.Instructions {

		if instruction.Member != nil {
			member := instruction.Member
			if member.Type == "participant" || member.Type == "actor" {
				participants[member.Name] = true
			}
		}

		if instruction.Life != nil {
			life := instruction.Life
			if life.Type == "create" {
				participants[life.Name] = true
			}
		}

		if instruction.Message != nil {
			message := instruction.Message
			switch message.Type {
			case "->>":
				// Caller calls Callee
				//caller := message.Left
				callee := message.Right
				methodName := message.Name
				_, calleeIsParticipant := participants[callee]

				cClass, cMethod := getCurrentContext()

				if cMethod == nil {
					// First call sets initial context
					methodClass := findClass(classes, callee)
					if methodClass == nil && !calleeIsParticipant {
						methodClass = findOrCreateDummyClass("AssumedClass")
					} else if methodClass == nil && calleeIsParticipant {
						methodClass = findOrCreateDummyClass(callee)
					}

					methodObj := findMethod(methodClass, methodName)
					if methodObj == nil {
						newMethod := generator.Method{
							AccessModifier: "public",
							Name:           methodName,
							ReturnType:     "void",
							Parameters:     []generator.Attribute{},
							MethodBody:     []generator.Body{},
						}
						methodClass.Methods = append(methodClass.Methods, newMethod)
						methodObj = &methodClass.Methods[len(methodClass.Methods)-1]
					}
					pushContext(methodClass, methodObj)
					// No caller here yet (top-level call), so no line is added.
				} else {
					// Add the call line to the current (caller) method
					callerClass, callerMethod := cClass, cMethod

					calleeClass := findClass(classes, callee)
					if calleeClass == nil && !calleeIsParticipant {
						calleeClass = findOrCreateDummyClass("AssumedClass")
					} else if calleeClass == nil && calleeIsParticipant {
						calleeClass = findOrCreateDummyClass(callee)
					}

					calleeMethod := findMethod(calleeClass, methodName)
					if calleeMethod == nil {
						newMethod := generator.Method{
							AccessModifier: "public",
							Name:           methodName,
							ReturnType:     "void",
							Parameters:     []generator.Attribute{},
							MethodBody:     []generator.Body{},
						}
						calleeClass.Methods = append(calleeClass.Methods, newMethod)
						calleeMethod = &calleeClass.Methods[len(calleeClass.Methods)-1]
					}

					// Determine call params
					callParams := []generator.Attribute{}
					for i, p := range message.Parameters {
						typ := "String"
						if i < len(calleeMethod.Parameters) {
							typ = calleeMethod.Parameters[i].Type
						}
						callParams = append(callParams, generator.Attribute{Name: p, Type: typ})
					}

					returnType := calleeMethod.ReturnType
					isVariable := returnType != "" && returnType != "void"

					objectRef := callee
					if calleeClass != nil {
						objectRef = ensureObjectReference(callerMethod, callerClass, calleeClass.ClassName)
					} else {
						objectRef = strings.ToLower(string(callee[0])) + callee[1:]
					}

					callBody := generator.Body{
						IsVariable:        isVariable,
						ObjFuncParameters: callParams,
						FunctionName:      objectRef + "." + methodName,
					}
					if isVariable {
						callBody.Variable = generator.Attribute{
							Name: "temp" + capitalize(methodName),
							Type: returnType,
						}
					}

					callerMethod.MethodBody = append(callerMethod.MethodBody, callBody)
					callLineIndex := len(callerMethod.MethodBody) - 1

					// Store call info for later renaming
					callReturnStack = append(callReturnStack, callReturnInfo{
						callerClass:  callerClass,
						callerMethod: callerMethod,
						lineIndex:    callLineIndex,
					})

					lastObjectIsConstructor = false
					lastObjectTempVarName = ""

					// Now push callee context
					pushContext(calleeClass, calleeMethod)
				}

			case "-->>":
				// Return from a method
				_, currentMethod := getCurrentContext()
				if currentMethod == nil {
					fmt.Println("No current function. Skipping return assignment.")
					continue
				}

				if lastObjectIsConstructor {
					for i, body := range currentMethod.MethodBody {
						if body.IsObjectCreation && body.ObjectName == lastObjectTempVarName {
							currentMethod.MethodBody[i].ObjectName = message.Name
							break
						}
					}
					lastObjectIsConstructor = false
					lastObjectTempVarName = ""
				} else {
					// Rename the variable in the caller method
					if len(callReturnStack) == 0 {
						fmt.Printf("No previous method call to assign return value: %s. Could not rename.\n", message.Name)
						// We don't retrofit now because we rely on the call stack.
						// If needed, we could fallback to a retrofit, but better to be explicit.
					} else {
						lastCall := callReturnStack[len(callReturnStack)-1]
						callReturnStack = callReturnStack[:len(callReturnStack)-1]

						callerMethod := lastCall.callerMethod
						if lastCall.lineIndex >= 0 && lastCall.lineIndex < len(callerMethod.MethodBody) {
							callLine := &callerMethod.MethodBody[lastCall.lineIndex]
							callLine.IsVariable = true
							// Rename to the returned name
							if callLine.Variable.Name == "" {
								callLine.Variable = generator.Attribute{
									Name: message.Name,
									Type: "String",
								}
							} else {
								callLine.Variable.Name = message.Name
							}
						} else {
							fmt.Printf("Couldn't assign return value name for method. Index out of range.\n")
						}
					}
				}

				popContext()
			}
		}
	}

	fmt.Println("TransformSequenceDiagram: completed transformation")
	return nil
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
// Helper to find an attribute in the current class by type
func findAttributeByType(class *generator.Class, typ string) *generator.Attribute {
	for i, attr := range class.Attributes {
		if attr.Type == typ {
			return &class.Attributes[i]
		}
	}
	return nil
}

// Helper to ensure an object reference exists for a given class type
func ensureObjectReference(
	currentFunction *generator.Method,
	currentFunctionClass *generator.Class,
	targetClassName string,
) string {
	// First, try to find an attribute in the class
	attr := findAttributeByType(currentFunctionClass, targetClassName)
	if attr != nil {
		// Found existing attribute, use it
		return attr.Name
	}

	// If not found as an attribute, check if we have already created one in the method
	// For simplicity, let's assume we haven't. We create a local variable:
	localVarName := strings.ToLower(string(targetClassName[0])) + targetClassName[1:] // e.g. AuthService -> authService

	// Check if already created in MethodBody
	for _, bodyLine := range currentFunction.MethodBody {
		if bodyLine.IsObjectCreation && bodyLine.ObjectType == targetClassName {
			return bodyLine.ObjectName // Already created
		}
	}

	// Not found, create a new line in the method body at the start:
	newBodyLine := generator.Body{
		IsObjectCreation: true,
		ObjectName:       localVarName,
		ObjectType:       targetClassName,
	}

	// Insert this creation at the beginning of the method body
	currentFunction.MethodBody = append([]generator.Body{newBodyLine}, currentFunction.MethodBody...)

	return localVarName
}

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
