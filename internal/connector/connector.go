package connector

import (
	"errors"
	"fmt"
	"strings"

	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
)

// TransformClassDiagram processes the class diagram and returns the classes and interfaces
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

			// Initialize class if it doesn't exist
			if _, exists := classes[className]; !exists {
				classes[className] = &generator.Class{
					ClassName:   className,
					Abstraction: []string{},
					Inherits:    "",
					Attributes:  []generator.Attribute{},
					Methods:     []generator.Method{},
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
				classes[className].Attributes = append(classes[className].Attributes, attr)

			} else if member.Operation != nil {
				method := generator.Method{
					AccessModifier: parseVisibility(member.Visibility),
					Name:           member.Operation.Name,
					IsStatic:       false,
					ReturnType:     member.Operation.Return,
					Parameters:     []generator.Attribute{},
					MethodBody:     []generator.Body{},
				}

				// Add parameters
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

	// Generate getters and setters for private attributes
	for _, class := range classes {
		generateGettersSetters(class)
	}

	return classes, interfaces, nil
}

// TransformSequenceDiagram processes the sequence diagram and updates the classes accordingly
func TransformSequenceDiagram(
	sequenceDiagram *reader.SequenceDiagram,
	classes map[string]*generator.Class,
	classTemplatePath, outputDir string,
) error {
	if sequenceDiagram == nil {
		return fmt.Errorf("sequence diagram is nil")
	}

	// Map of participant names to class names (for actors and participants)
	participants := make(map[string]string)
	methodContexts := make(map[string]*generator.Method) // Current method being processed for each class

	// Stack to keep track of method calls and their corresponding return variables
	type CallContext struct {
		caller     string
		callee     string
		methodName string
	}
	var callStack []CallContext

	// Map to track declared variables in each method context
	variableDeclarations := make(map[string]map[string]string) // methodName -> variableName -> type

	for _, instruction := range sequenceDiagram.Instructions {
		switch {
		case instruction.Member != nil:
			member := instruction.Member
			participants[member.Name] = member.Name // Map participant name to class name

		case instruction.Message != nil:
			message := instruction.Message

			sender := message.Left
			receiver := message.Right
			methodName := strings.TrimSuffix(message.Name, "()")

			// Ensure participants are in the map
			if _, exists := participants[sender]; !exists && !isActor(sender, sequenceDiagram) {
				participants[sender] = sender
				// Initialize the class if it doesn't exist
				if _, exists := classes[sender]; !exists {
					classes[sender] = &generator.Class{
						ClassName:  sender,
						Attributes: []generator.Attribute{},
						Methods:    []generator.Method{},
					}
				}
			}
			if _, exists := participants[receiver]; !exists && !isActor(receiver, sequenceDiagram) {
				participants[receiver] = receiver
				// Initialize the class if it doesn't exist
				if _, exists := classes[receiver]; !exists {
					classes[receiver] = &generator.Class{
						ClassName:  receiver,
						Attributes: []generator.Attribute{},
						Methods:    []generator.Method{},
					}
				}
			}

			// Determine if it's a return message
			isReturnMessage := strings.Contains(message.Type, "-->>") || strings.Contains(message.Type, "--x")

			if isReturnMessage {
				// Handle return messages by assigning the returned value to a variable
				if len(callStack) == 0 {
					return fmt.Errorf("call stack is empty, cannot process return message")
				}
				lastCall := callStack[len(callStack)-1]
				callStack = callStack[:len(callStack)-1]

				// If the caller is an actor, skip processing
				if isActor(lastCall.caller, sequenceDiagram) {
					// Actors do not have methods, skip
					continue
				}

				currentMethod := methodContexts[lastCall.caller]
				if currentMethod == nil {
					return fmt.Errorf("no method context for caller: %s", lastCall.caller)
				}

				// Update the last method call in the method body
				if len(currentMethod.MethodBody) == 0 {
					return fmt.Errorf("method body is empty for method %s in class %s", currentMethod.Name, lastCall.caller)
				}
				lastBodyIndex := len(currentMethod.MethodBody) - 1
				lastBody := &currentMethod.MethodBody[lastBodyIndex]

				// Get the called method to know the return type
				calleeClassName := participants[lastCall.callee]
				if calleeClassName == "" {
					if isActor(lastCall.callee, sequenceDiagram) {
						// Actors do not have classes, skip processing
						continue
					} else {
						return fmt.Errorf("participant %s not found in participants map", lastCall.callee)
					}
				}
				calleeClass := classes[calleeClassName]
				if calleeClass == nil {
					return fmt.Errorf("class %s not found", calleeClassName)
				}

				calledMethod := findOrCreateMethod(calleeClass, lastCall.methodName)
				returnType := calledMethod.ReturnType
				if returnType == "" || returnType == "void" {
					// No return value to assign
					continue
				}

				// Assign the return value to a variable
				returnVariable := message.Name // Use the message name as the variable name

				// Declare the variable if not already declared
				if _, exists := variableDeclarations[currentMethod.Name]; !exists {
					variableDeclarations[currentMethod.Name] = make(map[string]string)
				}
				if _, declared := variableDeclarations[currentMethod.Name][returnVariable]; !declared {
					variableDeclarations[currentMethod.Name][returnVariable] = returnType
					assignment := fmt.Sprintf("%s %s = %s;", returnType, returnVariable, strings.TrimSuffix(lastBody.FunctionName, ";"))
					lastBody.FunctionName = assignment
				} else {
					// Variable already declared, just assign
					assignment := fmt.Sprintf("%s = %s;", returnVariable, strings.TrimSuffix(lastBody.FunctionName, ";"))
					lastBody.FunctionName = assignment
				}

			} else {
				// Regular method call
				senderClassName := participants[sender]
				receiverClassName := participants[receiver]

				// Check if sender is an actor
				isSenderActor := isActor(sender, sequenceDiagram)

				if isSenderActor {
					// Start building method body for the method called by the actor
					receiverClass := classes[receiverClassName]
					if receiverClass == nil {
						return fmt.Errorf("class %s not found", receiverClassName)
					}

					// Find or create the method in the receiver class
					method := findOrCreateMethod(receiverClass, methodName)

					// Set the method context
					methodContexts[receiver] = method

					// Initialize variable declarations map for this method
					if _, exists := variableDeclarations[method.Name]; !exists {
						variableDeclarations[method.Name] = make(map[string]string)
					}

					// Push to call stack
					callStack = append(callStack, CallContext{
						caller:     sender,
						callee:     receiver,
						methodName: methodName,
					})

				} else {
					// Within a method, call another method on an attribute
					senderClass := classes[senderClassName]
					receiverClass := classes[receiverClassName]
					if senderClass == nil || receiverClass == nil {
						return fmt.Errorf("class not found for sender or receiver")
					}

					currentMethod := methodContexts[sender]
					if currentMethod == nil {
						return fmt.Errorf("no method context for sender: %s", sender)
					}

					// Initialize variable declarations map for this method if not present
					if _, exists := variableDeclarations[currentMethod.Name]; !exists {
						variableDeclarations[currentMethod.Name] = make(map[string]string)
					}

					// Find the attribute in the sender class that matches the receiver class
					var attributeName string
					for _, attr := range senderClass.Attributes {
						if attr.Type == receiverClass.ClassName {
							attributeName = attr.Name
							break
						}
					}
					if attributeName == "" {
						// If no attribute found, create a local variable
						attributeName = strings.ToLower(receiverClass.ClassName)
						if _, declared := variableDeclarations[currentMethod.Name][attributeName]; !declared {
							variableDeclarations[currentMethod.Name][attributeName] = receiverClass.ClassName
							// Add variable declaration to method body
							creationStatement := fmt.Sprintf("%s %s = new %s();", receiverClass.ClassName, attributeName, receiverClass.ClassName)
							currentMethod.MethodBody = append(currentMethod.MethodBody, generator.Body{
								FunctionName: creationStatement,
							})
						}
					}

					// Build the method call
					var functionCall string

					if methodName == receiverClass.ClassName {
						// Constructor call already handled
						continue
					} else {
						// Regular method call
						// Build argument list
						var arguments []string
						if len(message.Parameters) > 0 {
							for _, paramName := range message.Parameters {
								arguments = append(arguments, paramName)
							}
						} else {
							// Use parameter names from the method definition
							calledMethod := findOrCreateMethod(receiverClass, methodName)
							for _, param := range calledMethod.Parameters {
								arguments = append(arguments, param.Name)
							}
						}

						functionCall = fmt.Sprintf("%s.%s(%s);", attributeName, methodName, strings.Join(arguments, ", "))

						// Set the method context for the receiver if not already set
						if _, exists := methodContexts[receiver]; !exists {
							methodContexts[receiver] = findOrCreateMethod(receiverClass, methodName)
						}
					}

					// Add the method call to the current method's body
					body := generator.Body{
						FunctionName: functionCall,
					}

					currentMethod.MethodBody = append(currentMethod.MethodBody, body)

					// Push to call stack
					callStack = append(callStack, CallContext{
						caller:     sender,
						callee:     receiver,
						methodName: methodName,
					})
				}
			}

		case instruction.Life != nil && instruction.Life.Type == "create":
			// Handle object creation
			className := instruction.Life.Name
			participantName := className
			participants[participantName] = className

			// Ensure the class exists
			if _, exists := classes[className]; !exists {
				classes[className] = &generator.Class{
					ClassName:  className,
					Attributes: []generator.Attribute{},
					Methods:    []generator.Method{},
				}
			}

			// Determine the creator
			creatorName := instruction.Life.On
			if creatorName == "" || creatorName == "participant" || creatorName == "actor" {
				// Use the current method context
				// Find the last method context that is not an actor
				for i := len(callStack) - 1; i >= 0; i-- {
					if !isActor(callStack[i].caller, sequenceDiagram) {
						creatorName = callStack[i].caller
						break
					}
				}
				if creatorName == "" {
					return fmt.Errorf("cannot determine creator for object creation")
				}
			}

			// If the creator is an actor, we cannot proceed
			if isActor(creatorName, sequenceDiagram) {
				// Actors do not have methods to add object creation, skip
				continue
			}

			creatorClassName := participants[creatorName]
			if creatorClassName == "" {
				return fmt.Errorf("creator participant %s not found in participants map", creatorName)
			}
			creatorClass := classes[creatorClassName]
			if creatorClass == nil {
				return fmt.Errorf("creator class %s not found", creatorClassName)
			}

			// Get the creator's current method
			currentMethod := methodContexts[creatorName]
			if currentMethod == nil {
				return fmt.Errorf("no method context for creator: %s", creatorName)
			}

			// Initialize variable declarations map for this method if not present
			if _, exists := variableDeclarations[currentMethod.Name]; !exists {
				variableDeclarations[currentMethod.Name] = make(map[string]string)
			}

			// Build object creation statement as a local variable
			variableName := strings.ToLower(className)
			// Declare the variable if not already declared
			if _, declared := variableDeclarations[currentMethod.Name][variableName]; !declared {
				variableDeclarations[currentMethod.Name][variableName] = className
				creationStatement := fmt.Sprintf("%s %s = new %s();", className, variableName, className)
				body := generator.Body{
					FunctionName: creationStatement,
				}
				currentMethod.MethodBody = append(currentMethod.MethodBody, body)
			}

			// Optionally, you can keep track of the variable for future use

		}
	}

	return nil
}

// findOrCreateMethod searches for a method in the class and creates it if not found
func findOrCreateMethod(class *generator.Class, methodName string) *generator.Method {
	for i, method := range class.Methods {
		if method.Name == methodName {
			return &class.Methods[i]
		}
	}
	// Method not found, create it
	newMethod := generator.Method{
		AccessModifier: "public",
		Name:           methodName,
		ReturnType:     "void", // Default return type
		Parameters:     []generator.Attribute{},
		MethodBody:     []generator.Body{},
	}
	class.Methods = append(class.Methods, newMethod)
	return &class.Methods[len(class.Methods)-1]
}

// generateGettersSetters generates getters and setters for private attributes
func generateGettersSetters(class *generator.Class) {
	for _, attr := range class.Attributes {
		if attr.AccessModifier != "public" {
			// Check for existing getter
			getterName := "get" + strings.Title(attr.Name)
			if !methodExists(class, getterName) {
				// Generate getter
				getter := generator.Method{
					AccessModifier: "public",
					Name:           getterName,
					ReturnType:     attr.Type,
					Parameters:     []generator.Attribute{},
					MethodBody: []generator.Body{
						{
							FunctionName: fmt.Sprintf("return %s;", attr.Name),
						},
					},
				}
				class.Methods = append(class.Methods, getter)
			}
			// Check for existing setter
			setterName := "set" + strings.Title(attr.Name)
			if !methodExists(class, setterName) {
				// Generate setter
				setter := generator.Method{
					AccessModifier: "public",
					Name:           setterName,
					ReturnType:     "void",
					Parameters: []generator.Attribute{
						{
							Name: attr.Name,
							Type: attr.Type,
						},
					},
					MethodBody: []generator.Body{
						{
							FunctionName: fmt.Sprintf("this.%s = %s;", attr.Name, attr.Name),
						},
					},
				}
				class.Methods = append(class.Methods, setter)
			}
		}
	}
}

// methodExists checks if a method with the given name exists in the class
func methodExists(class *generator.Class, methodName string) bool {
	for _, method := range class.Methods {
		if method.Name == methodName {
			return true
		}
	}
	return false
}

// ensureConstructor checks if a default constructor exists and creates one if not
func ensureConstructor(class *generator.Class) {
	for _, method := range class.Methods {
		if method.Name == class.ClassName && len(method.Parameters) == 0 {
			// Default constructor already exists
			return
		}
	}
	// Create default constructor
	constructor := generator.Method{
		AccessModifier: "public",
		Name:           class.ClassName,
		ReturnType:     "",
		Parameters:     []generator.Attribute{},
		MethodBody:     []generator.Body{},
	}
	class.Methods = append([]generator.Method{constructor}, class.Methods...)
}

// isActor determines if a participant is an actor (not a participant class)
func isActor(name string, sequenceDiagram *reader.SequenceDiagram) bool {
	for _, instruction := range sequenceDiagram.Instructions {
		if instruction.Member != nil && instruction.Member.Type == "actor" && instruction.Member.Name == name {
			return true
		}
	}
	return false
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
		return "private"
	}
}
