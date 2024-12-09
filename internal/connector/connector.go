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
					} else if methodClass == nil {
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
					} else if calleeClass == nil {
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

					callBody := generator.Body{}
					if calleeMethod.Name == calleeClass.ClassName {
						callBody = generator.Body{
							IsObjectCreation:  true,
							ObjectName:        strings.ToLower(methodName),
							ObjectType:        calleeClass.ClassName,
							IsVariable:        isVariable,
							ObjFuncParameters: callParams,
							FunctionName:      objectRef + "." + methodName,
						}
					} else {
						callBody = generator.Body{
							IsVariable:        isVariable,
							ObjFuncParameters: callParams,
							FunctionName:      objectRef + "." + methodName,
						}
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
					// If the last object was created during the constructor, rename it
					for i, body := range currentMethod.MethodBody {
						if body.IsObjectCreation && body.ObjectName == lastObjectTempVarName {
							currentMethod.MethodBody[i].ObjectName = message.Name
							break
						}
					}
					lastObjectIsConstructor = false
					lastObjectTempVarName = ""
				} else {
					// Rename the variable in the caller method and set it as the return value
					if len(callReturnStack) == 0 {
						fmt.Printf("No previous method call to assign return value: %s. Could not rename.\n", message.Name)
						// No call stack entry means we can't assign return value to a previous call.
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
									Type: "String", // If you know the return type, adjust this accordingly
								}
							} else {
								callLine.Variable.Name = message.Name
							}

							// Set the caller method to return the newly named variable instead of the default placeholder
							callerMethod.ReturnValue = message.Name
						} else {
							fmt.Printf("Couldn't assign return value name for method. Index out of range.\n")
						}
					}
				}

				popContext()
			}
		}
	}

	// After processing the entire sequence diagram, fix return values for methods without return variables.
	for _, cls := range classes {
		for m := range cls.Methods {
			method := &cls.Methods[m]
			if method.ReturnType == "" || method.ReturnType == "void" {
				continue
			}

			// Check if there's a ReturnValue
			if method.ReturnValue == "" {
				continue // If no return value is set, skip
			}

			returnVar := method.ReturnValue
			returnType := method.ReturnType
			var existingVar *generator.Attribute
			var conflictingVar bool

			// Check for variable declarations or creations in the method body
			for _, bodyLine := range method.MethodBody {
				if bodyLine.IsObjectCreation && bodyLine.ObjectName == returnVar {
					if bodyLine.ObjectType == returnType {
						existingVar = &generator.Attribute{
							Name: returnVar,
							Type: returnType,
						}
					} else {
						conflictingVar = true
					}
					break
				}
				if bodyLine.IsDeclaration && bodyLine.Variable.Name == returnVar {
					if bodyLine.Variable.Type == returnType {
						existingVar = &bodyLine.Variable
					} else {
						conflictingVar = true
					}
					break
				}
			}

			// Handle conflicts or missing declarations
			if conflictingVar {
				// Create a new variable with a unique name
				uniqueVarName := returnVar + "Result"
				if isPrimitiveType(returnType) {
					// For primitives and String, declare and initialize with default value
					declBody := generator.Body{
						IsDeclaration: true,
						Variable: generator.Attribute{
							Name:  uniqueVarName,
							Type:  returnType,
							Value: defaultZero(returnType),
						},
					}
					method.MethodBody = append([]generator.Body{declBody}, method.MethodBody...)
				} else {
					// Non-primitive: create a new object
					creationBody := generator.Body{
						IsObjectCreation: true,
						ObjectName:       uniqueVarName,
						ObjectType:       returnType,
					}
					method.MethodBody = append([]generator.Body{creationBody}, method.MethodBody...)
				}
				// Update the method to return the unique variable
				method.ReturnValue = uniqueVarName
			} else if existingVar == nil {
				// No variable exists, create a new one
				if isPrimitiveType(returnType) {
					// For primitives and String, declare and initialize with default value
					declBody := generator.Body{
						IsDeclaration: true,
						Variable: generator.Attribute{
							Name:  returnVar,
							Type:  returnType,
							Value: defaultZero(returnType),
						},
					}
					method.MethodBody = append([]generator.Body{declBody}, method.MethodBody...)
				} else {
					// Non-primitive: create a new object
					creationBody := generator.Body{
						IsObjectCreation: true,
						ObjectName:       returnVar,
						ObjectType:       returnType,
					}
					method.MethodBody = append([]generator.Body{creationBody}, method.MethodBody...)
				}
			}
		}
	}
	finalizeVariableDeclarations(classes)

	fmt.Println("TransformSequenceDiagram: completed transformation")
	return nil
}

// Helper functions
func finalizeVariableDeclarations(classes map[string]*generator.Class) {
	for _, cls := range classes {
		for m := range cls.Methods {
			method := &cls.Methods[m]

			// existingVars maps the originally encountered variable name to a struct containing:
			// - The original type of the variable when first declared
			// - The final chosen name of the variable to use in code
			type varInfo struct {
				varType   string
				finalName string
			}

			existingVars := make(map[string]varInfo)

			for i, bodyLine := range method.MethodBody {
				var varName string
				var varType string
				var assignmentExpr string

				// Extract variable information depending on line type
				if bodyLine.IsDeclaration {
					varName = bodyLine.Variable.Name
					varType = bodyLine.Variable.Type
					if bodyLine.Variable.Value != nil {
						assignmentExpr = fmt.Sprintf("%v", bodyLine.Variable.Value)
					} else {
						assignmentExpr = ""
					}
				} else if bodyLine.IsObjectCreation {
					varName = bodyLine.ObjectName
					varType = bodyLine.ObjectType
					assignmentExpr = fmt.Sprintf("new %s(", varType)
					for pIndex, param := range bodyLine.ObjFuncParameters {
						if pIndex > 0 {
							assignmentExpr += ", "
						}
						if param.Value != nil {
							assignmentExpr += fmt.Sprintf("%v", param.Value)
						} else {
							assignmentExpr += param.Name
						}
					}
					assignmentExpr += ")"
				} else if bodyLine.IsVariable {
					varName = bodyLine.Variable.Name
					varType = bodyLine.Variable.Type
					// The assignment expression is basically the FunctionName plus any parameters
					baseCall := bodyLine.FunctionName
					assignmentExpr = baseCall

				} else {
					// No variable line, skip
					method.MethodBody[i] = bodyLine
					continue
				}

				if varName == "" || varType == "" {
					// If we don't have a varName or varType, just set and continue
					method.MethodBody[i] = bodyLine
					continue
				}

				info, alreadyDeclared := existingVars[varName]
				if !alreadyDeclared {
					// First time we see this variable
					existingVars[varName] = varInfo{
						varType:   varType,
						finalName: varName, // finalName is the same at first declaration
					}
					// The line stays as a declaration or object creation as is
					method.MethodBody[i] = bodyLine
				} else {
					// Variable name seen before
					if info.varType == varType {
						// Same type: This is a re-assignment
						finalName := info.finalName
						// Convert this line into a reassignment line
						bodyLine.IsDeclaration = false
						bodyLine.IsObjectCreation = false
						bodyLine.IsVariable = false
						// Just varName = assignmentExpr
						bodyLine.Variable = generator.Attribute{}
						//bodyLine.ObjFuncParameters = []generator.Attribute{}
						bodyLine.FunctionName = fmt.Sprintf("%s = %s", finalName, assignmentExpr)
						// finalName doesn't change
						method.MethodBody[i] = bodyLine
					} else {
						// Different type: we must rename the new variable to avoid conflict
						// The original variable keeps its finalName, we rename this second declaration
						newName := varName /*+ "Temp"*/
						// Update existingVars for the new type+name combination
						existingVars[varName] = varInfo{
							varType:   varType,
							finalName: newName,
						}
						// Convert to reassignment line
						bodyLine.IsDeclaration = false
						bodyLine.IsObjectCreation = false
						bodyLine.IsVariable = false
						bodyLine.Variable = generator.Attribute{}
						bodyLine.ObjFuncParameters = []generator.Attribute{}
						bodyLine.FunctionName = fmt.Sprintf("%s = %s", newName, assignmentExpr)
						method.MethodBody[i] = bodyLine
					}
				}
			}
		}
	}
}

func isPrimitiveType(typeName string) bool {
	primitiveTypes := map[string]bool{
		"int":     true,
		"boolean": true,
		"double":  true,
		"float":   true,
	}
	return primitiveTypes[typeName]
}

func defaultZero(typeName string) any {
	switch typeName {
	case "int":
		return "0"
	case "boolean":
		return "false"
	case "double", "float":
		return "0.0"
	default:
		return nil // Objects don't have default values; they are initialized with `new`
	}
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
