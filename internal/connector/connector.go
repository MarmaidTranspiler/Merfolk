package connector

import (
	"errors"
	"fmt"
	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
	"path/filepath"
	"strings"
)

// TransformClassDiagram transforms a Mermaid class diagram into code structures (classes and interfaces)
// and then generates Java code for those interfaces.
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
		if instruction.Member == nil {
			continue
		}

		className := instruction.Member.Class
		isInterface := instruction.Member.Operation == nil

		// Ensure class or interface entry is created
		if isInterface {
			ensureInterfaceEntry(interfaces, className)
		} else {
			ensureClassEntry(classes, className)
		}

		if instruction.Member.Attribute != nil {
			processClassDiagramAttribute(instruction.Member, classes, interfaces, className, isInterface)
		} else if instruction.Member.Operation != nil {
			processClassDiagramOperation(instruction.Member, classes, interfaces, className)
		}
	}

	// Generate Java code for interfaces
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

func ensureInterfaceEntry(interfaces map[string]*generator.Interface, className string) {
	if _, exists := interfaces[className]; !exists {
		interfaces[className] = &generator.Interface{
			InterfaceName:      className,
			Inherits:           []string{},
			AbstractAttributes: []generator.Attribute{},
			AbstractMethods:    []generator.Method{},
		}
		fmt.Printf("TransformClassDiagram: created new interface entry for %s\n", className)
	}
}

func ensureClassEntry(classes map[string]*generator.Class, className string) {
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

func processClassDiagramAttribute(
	member *reader.ClassMember,
	classes map[string]*generator.Class,
	interfaces map[string]*generator.Interface,
	className string,
	isInterface bool,
) {
	attr := generator.Attribute{
		AccessModifier:  parseVisibility(member.Visibility),
		Name:            member.Attribute.Name,
		Type:            member.Attribute.Type,
		IsClassVariable: false,
		IsConstant:      false,
		Value:           fmt.Sprintf("new %s()", member.Attribute.Type),
	}

	if isInterface {
		interfaces[className].AbstractAttributes = append(interfaces[className].AbstractAttributes, attr)
		// Also add to class as a normal attribute to maintain logic from original code
		classes[className].Attributes = append(classes[className].Attributes, attr)
		fmt.Printf("TransformClassDiagram: added attribute %s to interface %s\n", attr.Name, className)
	} else {
		classes[className].Attributes = append(classes[className].Attributes, attr)
		fmt.Printf("TransformClassDiagram: added attribute %s to class %s\n", attr.Name, className)
	}
}

func processClassDiagramOperation(
	member *reader.ClassMember,
	classes map[string]*generator.Class,
	interfaces map[string]*generator.Interface,
	className string,
) {
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

		// If parameter type corresponds to an existing class, add a dependency attribute
		if _, exists := classes[param.Type]; exists {

			classVar := generator.Attribute{
				AccessModifier:  "private",
				Name:            fmt.Sprintf("%sInstance", param.Type),
				Type:            param.Type,
				IsClassVariable: false,
				IsConstant:      false,
				IsObject:        isPrimitiveType(param.Type),
				Value: func() string {
					if isPrimitiveType(param.Type) {
						return defaultZero(param.Type) // Replace with appropriate default
					}
					return fmt.Sprintf("new %s()", param.Type)
				}(),
			}
			classes[className].Attributes = append(classes[className].Attributes, classVar)
			fmt.Printf("TransformClassDiagram: added class dependency attribute %sInstance to class %s\n", param.Type, className)
		}
	}

	iface, ifaceExists := interfaces[className]
	cls, clsExists := classes[className]
	if ifaceExists && method.ReturnType == "" {
		// In interfaces, methods have no body, just abstract definition
		iface.AbstractMethods = append(iface.AbstractMethods, method)
		fmt.Printf("TransformClassDiagram: added abstract method %s to interface %s\n", method.Name, className)
	} else if clsExists {
		cls.Methods = append(cls.Methods, method)
		fmt.Printf("TransformClassDiagram: added method %s to class %s\n", method.Name, className)
	}
}

// TransformSequenceDiagram transforms a Mermaid sequence diagram into code instructions within the classes.
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

	// Conditional context for alt/if/else blocks
	type conditionalContext struct {
		active      bool
		ifCondition string
		ifBody      []generator.Body
		elseBody    []generator.Body
		seenElse    bool

		origClass  *generator.Class
		origMethod *generator.Method
	}
	var currentConditional conditionalContext

	// Helper functions for managing call stack context
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
		currentMethod *generator.Method,
		currentClass *generator.Class,
		targetClassName string,
	) string {
		if currentClass == nil || currentMethod == nil {
			return strings.ToLower(string(targetClassName[0])) + targetClassName[1:]
		}

		// Check existing attributes
		for i, attr := range currentClass.Attributes {
			if attr.Type == targetClassName {
				return currentClass.Attributes[i].Name
			}
		}

		// Check object creation lines in the method body
		for _, bodyLine := range currentMethod.MethodBody {
			if bodyLine.IsObjectCreation && bodyLine.ObjectType == targetClassName {
				return bodyLine.ObjectName
			}
		}

		// If no reference found, create a new local object
		localVarName := strings.ToLower(string(targetClassName[0])) + targetClassName[1:]
		newBodyLine := generator.Body{
			IsObjectCreation: true,
			ObjectName:       localVarName,
			ObjectType:       targetClassName,
		}
		// Prepend object creation
		currentMethod.MethodBody = append([]generator.Body{newBodyLine}, currentMethod.MethodBody...)
		return localVarName
	}

	findClass := func(classes map[string]*generator.Class, className string) *generator.Class {
		if c, exists := classes[className]; exists {
			return c
		}
		return nil
	}

	findMethod := func(class *generator.Class, methodName string) *generator.Method {
		for i, m := range class.Methods {
			if m.Name == methodName {
				return &class.Methods[i]
			}
		}
		return nil
	}

	finalizeConditionalBlock := func(m *generator.Method) {
		if !currentConditional.active {
			return
		}
		condBody := generator.Body{
			IsCondition: true,
			Condition:   currentConditional.ifCondition,
			IfBody:      currentConditional.ifBody,
			ElseBody:    currentConditional.elseBody,
		}
		m.MethodBody = append(m.MethodBody, condBody)
		currentConditional = conditionalContext{}
	}

	startIfBlock := func(definition []string) {
		currentConditional.active = true
		currentConditional.ifCondition = strings.Join(definition, " ")
	}

	startElseBlock := func(definition []string) {
		// Treat any subsequent alt as else block
		currentConditional.seenElse = true
	}

	addInstructionToCurrentContext := func(b generator.Body) {
		_, m := getCurrentContext()
		if m == nil {
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

	// Process the instructions of the sequence diagram
	for _, instruction := range sequenceDiagram.Instructions {
		// Handle members/participants
		if instruction.Member != nil && (instruction.Member.Type == "participant" || instruction.Member.Type == "actor") {
			participants[instruction.Member.Name] = true
		}

		// Handle life creation
		if instruction.Life != nil && instruction.Life.Type == "create" {
			participants[instruction.Life.Name] = true
		}

		// Handle alt/if/else conditions
		if instruction.Alt != nil {
			alt := instruction.Alt
			currClass, currMethod := getCurrentContext()
			if currMethod == nil {
				fmt.Println("Warning: 'alt' encountered outside of any method context. Ignoring.")
				continue
			}
			if !currentConditional.active {
				// Start a new if block
				startIfBlock(alt.Definition)
				currentConditional.origClass = currClass
				currentConditional.origMethod = currMethod
			} else {
				// If block already active, this becomes an else block
				if !currentConditional.seenElse {

					startElseBlock(alt.Definition)
				} else {
					fmt.Println("Warning: Multiple else blocks not supported. Ignoring extra alt.")
				}
			}
			continue
		}

		if instruction.Else != nil {
			if !currentConditional.active {
				fmt.Println("Warning: 'else' encountered without an active 'alt' block. Ignoring.")
				continue
			}
			if currentConditional.seenElse {
				fmt.Println("Warning: Multiple 'else' blocks encountered. Ignoring extra 'else'.")
				continue
			}
			// Switch to else block
			currentConditional.seenElse = true
			continue
		}

		if instruction.End != nil {
			if currentConditional.active && currentConditional.origMethod != nil {
				finalizeConditionalBlock(currentConditional.origMethod)
			} else {
				fmt.Println("Warning: 'end' encountered without an active 'alt' block or method context.")
			}
			continue
		}

		// Loops and switches not fully implemented, skip
		if instruction.Loop != nil || instruction.Switch != nil {
			continue
		}

		if instruction.Message != nil {
			message := instruction.Message
			switch message.Type {
			case "->>":
				// Caller calls Callee
				callLineIndex := 0
				cClass, cMethod := getCurrentContext()

				calleeName := message.Right
				_, calleeIsParticipant := participants[calleeName]

				if cMethod == nil {
					// First call: set initial context
					calleeClass := findClass(classes, calleeName)
					if calleeClass == nil && !calleeIsParticipant {
						calleeClass = findOrCreateDummyClass("AssumedClass")
					} else if calleeClass == nil {
						calleeClass = findOrCreateDummyClass(calleeName)
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

					callReturnStack = append(callReturnStack, callReturnInfo{
						callerClass:  calleeClass,
						callerMethod: calleeMethod,
						lineIndex:    callLineIndex,
					})

					pushContext(calleeClass, calleeMethod)
				} else {
					// Normal call from caller to callee
					callerClass, callerMethod := cClass, cMethod
					calleeClass := findClass(classes, calleeName)
					if calleeClass == nil && !calleeIsParticipant {
						calleeClass = findOrCreateDummyClass("AssumedClass")
					} else if calleeClass == nil {
						calleeClass = findOrCreateDummyClass(calleeName)
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

					// Prepare parameters for the call
					callParams := []generator.Attribute{}
					for i, p := range message.Parameters {
						typ := "String"
						if i < len(calleeMethod.Parameters) {
							typ = calleeMethod.Parameters[i].Type
						}
						callParams = append(callParams, generator.Attribute{Name: p, Type: typ})
					}

					//fmt.Println(callerMethod.Name, "   ", callerMethod.ReturnType)

					returnType := calleeMethod.ReturnType
					isVariable := returnType != "" && returnType != "void"
					isConstructor := calleeMethod.Name == calleeClass.ClassName
					// fmt.Println(message.Name, returnType, isVariable, isConstructor)

					objectRef := ensureObjectReference(callerMethod, callerClass, calleeClass.ClassName)

					var callBody generator.Body
					if calleeMethod.Name == calleeClass.ClassName {
						// Constructor call
						callBody = generator.Body{
							IsObjectCreation:  true,
							ObjectName:        strings.ToLower(message.Name),
							ObjectType:        calleeClass.ClassName,
							IsVariable:        isVariable,
							ObjFuncParameters: callParams,
							FunctionName:      objectRef + "." + message.Name,
						}
					} else {
						// Regular method call
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

					callLineIndex = len(cMethod.MethodBody) - 1

					pushContext(calleeClass, calleeMethod)

					if !isVariable && !isConstructor {
						popContext()
					} else {
						callReturnStack = append(callReturnStack, callReturnInfo{
							callerClass:  callerClass,
							callerMethod: callerMethod,
							lineIndex:    callLineIndex,
						})
					}

				}

			case "-->>":
				// Return from a method
				currentClass, currentMethod := getCurrentContext()

				if currentMethod == nil {
					fmt.Println("No current function. Skipping return assignment.")
					continue
				} else {

					if len(callReturnStack) == 0 {
						fmt.Printf("No previous method call to assign return value: %s. Could not rename.\n", message.Name)
					} else {
						lastCall := callReturnStack[len(callReturnStack)-1]
						callReturnStack = callReturnStack[:len(callReturnStack)-1]

						callerMethod := lastCall.callerMethod

						if lastCall.lineIndex >= 0 && lastCall.lineIndex < len(callerMethod.MethodBody) {
							callLine := &callerMethod.MethodBody[lastCall.lineIndex]

							//fmt.Println("find check", callLine.FunctionName, message.Left, findMethod(findClass(classes, message.Left), extractAfterLastDot(callLine.FunctionName)).Name)
							//fmt.Println(callLine.Variable.Name, message.Name)

							callLine.IsVariable = true
							if callLine.Variable.Name == "" {

								callLine.Variable = generator.Attribute{
									Name: message.Name,
									Type: "TEMP",
								}
							} else if message.Left == currentClass.ClassName {
								//fmt.Println(callLine.Variable.Name, message.Name)

								if findMethod(findClass(classes, message.Left), extractAfterLastDot(callLine.FunctionName)) != nil {
									fmt.Println(callLine.Variable.Name, message.Name)
									callLine.Variable.Name = message.Name
								}

								//fmt.Println(callLine.Variable.Name, message.Name)

							}

							currentMethod.ReturnValue = message.Name
						} else {

							fmt.Println("Couldn't assign return value name for method. Index out of range.", callerMethod.Name, callerMethod.ReturnValue, currentMethod.Name, currentClass)
						}
					}
				}

				popContext()
			}
		}
	}

	// If there's any open conditional block at the end, finalize it
	_, finalMethod := getCurrentContext()
	if currentConditional.active && finalMethod != nil {
		finalizeConditionalBlock(finalMethod)
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

	// After processing the entire sequence diagram, fix variable declarations
	finalizeVariableDeclarations(classes)

	if len(callStack) > 0 {
		fmt.Printf("Warning: Stack not empty after processing. Remaining size: %d\n", len(callStack))
		for _, ctx := range callStack {
			fmt.Printf("Unfinished context: class=%s, method=%s\n", ctx.class.ClassName, ctx.method.Name)
		}
	}

	fmt.Println("TransformSequenceDiagram: completed transformation")
	return nil
}

// finalizeVariableDeclarations ensures variables are properly declared and assigned
// without changing the logic. It attempts to tidy up variable usage in method bodies.
func finalizeVariableDeclarations(classes map[string]*generator.Class) {
	for _, cls := range classes {
		for m := range cls.Methods {
			method := &cls.Methods[m]

			type varInfo struct {
				varType   string
				finalName string
			}
			existingVars := make(map[string]varInfo)

			for i, bodyLine := range method.MethodBody {
				varName, varType, assignmentExpr := extractVariableInfo(bodyLine)

				if varName == "" || varType == "" {
					continue
				}

				info, alreadyDeclared := existingVars[varName]
				if !alreadyDeclared {
					// First time we see this variable
					existingVars[varName] = varInfo{
						varType:   varType,
						finalName: varName,
					}
					method.MethodBody[i] = bodyLine
				} else {
					// Variable name seen before
					if info.varType == varType {
						// Same type: re-assignment
						method.MethodBody[i] = convertToReassignment(bodyLine, info.finalName, assignmentExpr)
					} else {
						// Different type: rename to avoid conflict
						newName := varName // simply reuse same name as finalName
						existingVars[varName] = varInfo{
							varType:   varType,
							finalName: newName,
						}
						method.MethodBody[i] = convertToReassignment(bodyLine, newName, assignmentExpr)
					}
				}
			}
		}
	}
}

// extractVariableInfo extracts variable details from a Body line
func extractVariableInfo(bodyLine generator.Body) (varName, varType, assignmentExpr string) {
	if bodyLine.IsDeclaration {
		varName = bodyLine.Variable.Name
		varType = bodyLine.Variable.Type
		if bodyLine.Variable.Value != nil {
			assignmentExpr = fmt.Sprintf("%v", bodyLine.Variable.Value)
		}
	} else if bodyLine.IsObjectCreation {
		varName = bodyLine.ObjectName
		varType = bodyLine.ObjectType
		assignmentExpr = buildObjectCreationString(varType, bodyLine.ObjFuncParameters)
	} else if bodyLine.IsVariable && bodyLine.FunctionName != "" {
		// Variable assignment from a function call
		varName = bodyLine.Variable.Name
		varType = bodyLine.Variable.Type
		assignmentExpr = bodyLine.FunctionName
	}
	return
}

// buildObjectCreationString constructs a string for object creation with parameters
func buildObjectCreationString(varType string, params []generator.Attribute) string {
	var sb strings.Builder
	sb.WriteString("new ")
	sb.WriteString(varType)
	sb.WriteString("(")
	for i, p := range params {
		if i > 0 {
			sb.WriteString(", ")
		}
		if p.Value != nil {
			sb.WriteString(fmt.Sprintf("%v", p.Value))
		} else {
			sb.WriteString(p.Name)
		}
	}
	sb.WriteString(")")
	return sb.String()
}

func extractAfterLastDot(input string) string {
	if idx := strings.LastIndex(input, "."); idx != -1 {
		return input[idx+1:] // Return substring after the last dot
	}
	return input // If no dot, return the original string
}

// convertToReassignment modifies a body line to represent a reassignment instead of a declaration
func convertToReassignment(bodyLine generator.Body, finalName, assignmentExpr string) generator.Body {
	bodyLine.IsDeclaration = false
	bodyLine.IsObjectCreation = false
	bodyLine.IsVariable = false
	//bodyLine.Variable = generator.Attribute{}
	//bodyLine.ObjFuncParameters = []generator.Attribute{}
	bodyLine.FunctionName = fmt.Sprintf("%s = %s", finalName, assignmentExpr)
	return bodyLine
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

func isPrimitiveType(typeName string) bool {
	primitiveTypes := map[string]bool{
		"int":     true,
		"boolean": true,
		"double":  true,
		"float":   true,
	}
	return primitiveTypes[typeName]
}

func defaultZero(typeName string) string {
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
		return "null"
	}
}

func capitalize(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(string(str[0])) + str[1:]
}
