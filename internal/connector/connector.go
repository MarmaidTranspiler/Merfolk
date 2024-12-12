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

	findClass := func(classes map[string]*generator.Class, className string) *generator.Class {
		if class, exists := classes[className]; exists {
			return class
		}
		return nil
	}

	findMethod := func(class *generator.Class, methodName string) *generator.Method {
		for i, method := range class.Methods {
			if method.Name == methodName {
				return &class.Methods[i]
			}
		}
		return nil
	}

	// A small structure to handle alt/else/end logic:
	type conditionalContext struct {
		// True if we're currently building an if block
		active bool
		// Condition for the if block (from the first alt line)
		ifCondition string
		// IfBody: holds instructions inside the 'if' portion
		ifBody []generator.Body
		// ElseBody: holds instructions inside the 'else' portion (if any)
		elseBody []generator.Body
		// Track whether we've seen an else block yet
		seenElse bool
	}

	var currentConditional conditionalContext

	// Helper to flush conditional context into the current method
	finalizeConditionalBlock := func(m *generator.Method) {
		fmt.Println("in finalise this is method:", m.Name)

		if !currentConditional.active {
			return
		}
		// Create a single Body with IsCondition = true
		condBody := generator.Body{
			IsCondition: true,
			Condition:   currentConditional.ifCondition,
			IfBody:      currentConditional.ifBody,
		}
		if currentConditional.seenElse {
			condBody.ElseBody = currentConditional.elseBody
		}
		m.MethodBody = append(m.MethodBody, condBody)
		// Reset the conditional context
		currentConditional = conditionalContext{}
	}

	// Add a helper to start if block from Alt definition
	startIfBlock := func(definition []string) {
		currentConditional.active = true
		condition := strings.Join(definition, " ")
		// If the line is just 'else', we treat it as else with no condition
		// but since it's the first block, that wouldn't make sense, so let's assume first alt is always condition
		currentConditional.ifCondition = condition
	}

	// Switch to else block when we encounter a second alt line inside an if block
	startElseBlock := func(definition []string) {
		// If definition includes "else" keyword or no suitable condition, treat as else
		condition := strings.Join(definition, " ")
		// For simplicity, any second alt line is treated as else block if "else" is present
		if strings.Contains(condition, "else") {
			currentConditional.seenElse = true
			// Now we store subsequent instructions into elseBody
		} else {
			// If no 'else' keyword, treat it as else anyway for simplicity
			currentConditional.seenElse = true
		}
	}

	// A helper to add a body line either to method or conditional context
	addInstructionToCurrentContext := func(b generator.Body) {
		_, m := getCurrentContext()
		if m == nil {
			// No method context, ignore or print warning
			fmt.Println("Warning: Instruction outside of method context, skipping line.")
			return
		}

		if currentConditional.active {
			if currentConditional.seenElse {
				currentConditional.elseBody = append(currentConditional.elseBody, b)
			} else {
				currentConditional.ifBody = append(currentConditional.ifBody, b)
			}
		} else {
			m.MethodBody = append(m.MethodBody, b)
		}
	}

	// Main loop
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

		// Handle Alt (if/else) blocks ALTBLOCK
		if instruction.Alt != nil {
			alt := instruction.Alt
			class, m := getCurrentContext()
			fmt.Println("ALT this is class:", class.ClassName)

			fmt.Println("ALT this is method:", m.Name)

			if m == nil {
				fmt.Println("Warning: alt encountered outside of any method context. Ignoring.")
				continue
			}

			if !currentConditional.active {
				// Start a new if block
				startIfBlock(alt.Definition)
			} else {
				// If block is active, this is either else block
				if !currentConditional.seenElse {
					// Start else block
					startElseBlock(alt.Definition)
				} else {
					// We've already got an else block.
					// For simplicity, ignore additional alt lines or treat them as no-ops
					fmt.Println("Warning: Multiple else blocks not supported. Ignoring extra alt.")
				}
			}

			continue
		}

		if instruction.End != nil {
			// End of alt/if block
			_, m := getCurrentContext()
			fmt.Println("END this is class:", m.Name)

			if currentConditional.active {
				finalizeConditionalBlock(m)
			}
			continue
		}

		if instruction.Loop != nil {
			// For now, loops are not implemented, just a placeholder
			// You could implement loop logic similarly to alt/else if needed.
			continue
		}

		if instruction.Switch != nil {
			// For now, switch (activate/deactivate) are not implemented
			// Just ignore or handle similarly if needed.
			continue
		}

		if instruction.Message != nil {
			message := instruction.Message
			switch message.Type {
			case "->>":
				// Caller calls Callee
				cClass, cMethod := getCurrentContext()

				callee := message.Right
				_, calleeIsParticipant := participants[callee]
				if cMethod == nil {
					// First call sets initial context
					methodClass := findClass(classes, callee)
					if methodClass == nil && !calleeIsParticipant {
						methodClass = findOrCreateDummyClass("AssumedClass")
					} else if methodClass == nil {
						methodClass = findOrCreateDummyClass(callee)
					}

					methodObj := findMethod(methodClass, message.Name)
					if methodObj == nil {
						newMethod := generator.Method{
							AccessModifier: "public",
							Name:           message.Name,
							ReturnType:     "void",
							Parameters:     []generator.Attribute{},
							MethodBody:     []generator.Body{},
						}
						methodClass.Methods = append(methodClass.Methods, newMethod)
						methodObj = &methodClass.Methods[len(methodClass.Methods)-1]
					}
					pushContext(methodClass, methodObj)
				} else {
					// Add the call line to the current (caller) method
					callerClass, callerMethod := cClass, cMethod

					calleeClass := findClass(classes, callee)
					if calleeClass == nil && !calleeIsParticipant {
						calleeClass = findOrCreateDummyClass("AssumedClass")
					} else if calleeClass == nil {
						calleeClass = findOrCreateDummyClass(callee)
					}

					calleeMethod := findMethod(calleeClass, message.Name)
					if calleeMethod == nil {
						newMethod := generator.Method{
							AccessModifier: "public",
							Name:           message.Name,
							ReturnType:     "void",
							Parameters:     []generator.Attribute{},
							MethodBody:     []generator.Body{},
						}
						calleeClass.Methods = append(calleeClass.Methods, newMethod)
						calleeMethod = &calleeClass.Methods[len(calleeClass.Methods)-1]
					}

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
							ObjectName:        strings.ToLower(message.Name),
							ObjectType:        calleeClass.ClassName,
							IsVariable:        isVariable,
							ObjFuncParameters: callParams,
							FunctionName:      objectRef + "." + message.Name,
						}
					} else {
						callBody = generator.Body{
							IsVariable:        isVariable,
							ObjFuncParameters: callParams,
							FunctionName:      objectRef + "." + message.Name,
						}
					}

					if isVariable {
						callBody.Variable = generator.Attribute{
							Name: "temp" + capitalize(message.Name),
							Type: returnType,
						}
					}

					addInstructionToCurrentContext(callBody)
					callLineIndex := -1
					if cMethod != nil {
						callLineIndex = len(cMethod.MethodBody) - 1
					}

					callReturnStack = append(callReturnStack, callReturnInfo{
						callerClass:  callerClass,
						callerMethod: callerMethod,
						lineIndex:    callLineIndex,
					})

					lastObjectIsConstructor = false
					lastObjectTempVarName = ""

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
					if len(callReturnStack) == 0 {
						fmt.Printf("No previous method call to assign return value: %s. Could not rename.\n", message.Name)
					} else {
						lastCall := callReturnStack[len(callReturnStack)-1]
						callReturnStack = callReturnStack[:len(callReturnStack)-1]

						callerMethod := lastCall.callerMethod
						if lastCall.lineIndex >= 0 && lastCall.lineIndex < len(callerMethod.MethodBody) {
							callLine := &callerMethod.MethodBody[lastCall.lineIndex]
							callLine.IsVariable = true
							if callLine.Variable.Name == "" {
								callLine.Variable = generator.Attribute{
									Name: message.Name,
									Type: "String",
								}
							} else {
								callLine.Variable.Name = message.Name
							}
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

	// If there's any open conditional block at the end (no 'end' found)
	// finalize it anyway.
	class, finalMethod := getCurrentContext()
	fmt.Println("this is class:", class.ClassName)
	fmt.Println("this is method:", finalMethod.Name)

	if currentConditional.active {
		finalizeConditionalBlock(finalMethod)
	}

	// After processing the entire sequence diagram, fix return values for methods
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
