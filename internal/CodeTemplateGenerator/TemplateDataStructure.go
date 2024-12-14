package CodeTemplateGenerator

// Class represents a class definition in code.
// It contains the class name, abstraction details, inheritance information, attributes, and methods.
type Class struct {
	ClassName   string      // The name of the class.
	Abstraction []string    // A list of abstractions or interfaces the class implements.
	Inherits    string      // The name of the class or interface from which this class inherits.
	Attributes  []Attribute // A list of attributes (fields) defined in the class.
	Methods     []Method    // A list of methods (functions) defined in the class.
}

// Attribute represents a class field or property.
// It holds details about its access level, type, initialization, and other metadata.
type Attribute struct {
	AccessModifier         string           // The access modifier of the attribute (e.g., public, private).
	Name                   string           // The name of the attribute.
	Type                   string           // The data type of the attribute (e.g., int, string).
	IsClassVariable        bool             // True if the attribute is a class-level variable (static).
	IsConstant             bool             // True if the attribute is a constant.
	IsAttributeInitialized bool             // True if the attribute is initialized at declaration.
	ObjectConstructorArgs  []ConstructorArg // Arguments for the constructor if the attribute represents an object.
	IsObject               bool             // True if the attribute represents an object.
	Value                  any              // The value of the attribute (can be any type).
}

// Body represents the body of a method or a logical block in code.
// It includes object creation, conditional logic, and variable declarations.
type Body struct {
	IsObjectCreation  bool        // True if this block involves creating an object.
	ObjectName        string      // The name of the object being created.
	ObjectType        string      // The type of the object being created.
	ObjFuncParameters []Attribute // Attributes passed as parameters during object creation.

	IsVariable   bool   // True if this block involves passing method results into a variable.
	FunctionName string // The name of the function being called.

	IsCondition bool   // True if this block represents a conditional statement.
	Condition   string // The condition being checked in the statement.
	IfBody      []Body // The body of the 'if' block.
	ElseBody    []Body // The body of the 'else' block.

	IsDeclaration bool      // True if this block involves a variable declaration or initialization.
	Variable      Attribute // The variable being declared or initialized.
}

// Method represents a method (function) in a class.
// It includes information about its access modifier, name, parameters, return type, and body.
type Method struct {
	AccessModifier string      // The access modifier of the method (e.g., public, private).
	Name           string      // The name of the method.
	IsStatic       bool        // True if the method is static.
	ReturnType     string      // The return type of the method.
	Parameters     []Attribute // A list of parameters that the method accepts.
	MethodBody     []Body      // The body of the method, consisting of multiple logic blocks.
	ReturnValue    string      // The return value of the method (if any).
}

// Interface represents an interface definition in code.
// It contains the interface name, inherited interfaces, and abstract attributes/methods.
type Interface struct {
	InterfaceName      string      // The name of the interface.
	Inherits           []string    // A list of interfaces this interface inherits from.
	AbstractAttributes []Attribute // A list of abstract attributes (properties) in the interface.
	AbstractMethods    []Method    // A list of abstract methods in the interface.
}

// ConstructorArg represents a single argument for a constructor.
// It includes the argument type and its value.
type ConstructorArg struct {
	Type  string // The data type of the argument.
	Value string // The value of the argument.
}
