package connector

import (
	_ "fmt"
	_ "strings"

	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
)

// mapVisibility maps Mermaid visibility symbols to Java access modifiers.
func mapVisibility(symbol string) string {
	switch symbol {
	case "+":
		return "public"
	case "-":
		return "private"
	case "#":
		return "protected"
	case "~":
		return "/* package-private */"
	default:
		return "public"
	}
}

// convertParameters converts parser parameters to generator attributes.
func convertParameters(params []*reader.Parameter) []generator.Attribute {
	var result []generator.Attribute
	for _, p := range params {
		result = append(result, generator.Attribute{
			Name: p.Name,
			Type: p.Type,
		})
	}
	return result
}

// getOrCreateClass retrieves or creates a JavaClass instance.
func getOrCreateClass(
	name string,
	classMap map[string]*generator.JavaClass,
) *generator.JavaClass {
	if cls, exists := classMap[name]; exists {
		return cls
	}
	cls := &generator.JavaClass{
		ClassName: name,
	}
	classMap[name] = cls
	return cls
}

// getOrCreateInterface retrieves or creates an InterfaceClass instance.
func getOrCreateInterface(
	name string,
	interfaceMap map[string]*generator.InterfaceClass,
) *generator.InterfaceClass {
	if iface, exists := interfaceMap[name]; exists {
		return iface
	}
	iface := &generator.InterfaceClass{
		InterfaceName: name,
	}
	interfaceMap[name] = iface
	return iface
}

// transformClassDiagram transforms a ClassDiagram into JavaClass and InterfaceClass instances.
func transformClassDiagram(
	classDiagram *reader.ClassDiagram,
) ([]generator.JavaClass, []generator.InterfaceClass) {
	classMap := make(map[string]*generator.JavaClass)
	interfaceMap := make(map[string]*generator.InterfaceClass)

	for _, instr := range classDiagram.Instructions {
		// Handle ClassMember
		if instr.Member != nil {
			className := instr.Member.Class
			visibility := mapVisibility(instr.Member.Visibility)

			if instr.Member.Attribute != nil {
				// Attribute
				attr := instr.Member.Attribute
				javaClass := getOrCreateClass(className, classMap)
				javaAttr := generator.Attribute{
					AccessModifier: visibility,
					Name:           attr.Name,
					Type:           attr.Type,
				}
				javaClass.Attributes = append(javaClass.Attributes, javaAttr)
			} else if instr.Member.Operation != nil {
				// Operation (Method)
				op := instr.Member.Operation
				if _, exists := interfaceMap[className]; exists {
					javaInterface := getOrCreateInterface(className, interfaceMap)
					javaMethod := generator.Method{
						AccessModifier: visibility,
						Name:           op.Name,
						Type:           op.Return,
						Parameters:     convertParameters(op.Parameters),
					}
					javaInterface.Methods = append(javaInterface.Methods, javaMethod)
				} else {
					javaClass := getOrCreateClass(className, classMap)
					javaMethod := generator.Method{
						AccessModifier: visibility,
						Name:           op.Name,
						Type:           op.Return,
						Parameters:     convertParameters(op.Parameters),
					}
					javaClass.Methods = append(javaClass.Methods, javaMethod)
				}
			}
		}

		// Handle Relationship
		if instr.Relationship != nil {
			rel := instr.Relationship
			leftClassName := rel.LeftClass
			rightClassName := rel.RightClass
			relationshipType := rel.Type

			switch relationshipType {
			case "<|--":
				// Inheritance
				subClass := getOrCreateClass(leftClassName, classMap)
				subClass.Inherits = rightClassName
			case "..|>":
				// Implementation
				subClass := getOrCreateClass(leftClassName, classMap)
				subClass.Implements = append(subClass.Implements, rightClassName)
			case "<|..":
				// Interface inheritance
				subInterface := getOrCreateInterface(leftClassName, interfaceMap)
				subInterface.Inherits = append(subInterface.Inherits, rightClassName)
			}
		}

		// Handle Annotation
		if instr.Annotation != nil {
			annotation := instr.Annotation
			className := annotation.Class
			if annotation.Name == "interface" {
				javaInterface := getOrCreateInterface(className, interfaceMap)
				// Transfer methods and attributes if any
				if javaClass, exists := classMap[className]; exists {
					javaInterface.Methods = append(
						javaInterface.Methods,
						javaClass.Methods...,
					)
					delete(classMap, className)
				}
			} else if annotation.Name == "abstract" {
				javaClass := getOrCreateClass(className, classMap)
				javaClass.Abstraction = "abstract"
			}
		}
	}

	// Collect classes and interfaces into slices
	var javaClasses []generator.JavaClass
	for _, cls := range classMap {
		javaClasses = append(javaClasses, *cls)
	}

	var javaInterfaces []generator.InterfaceClass
	for _, iface := range interfaceMap {
		javaInterfaces = append(javaInterfaces, *iface)
	}

	return javaClasses, javaInterfaces
}
