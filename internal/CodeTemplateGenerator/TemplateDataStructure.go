package CodeTemplateGenerator

type Class struct {
	ClassName   string
	Abstraction []string
	Inherits    string
	Attributes  []Attribute
	Methods     []Method
}

// Attribute IsClassVariable/ True, falls es sich um eine Klassenvariable handelt */
// Attribute IsConstant/ True, falls es sich um eine Konstante handelt
type Attribute struct {
	AccessModifier         string
	Name                   string
	Type                   string
	IsClassVariable        bool
	IsConstant             bool
	IsAttributeInitialized bool
	ObjectConstructorArgs  []ConstructorArg
	IsObject               bool
	Value                  any
}

type Body struct {
	// Object creation
	IsObjectCreation  bool
	ObjectName        string
	ObjectType        string
	ObjFuncParameters []Attribute
	// passing  method results into variable
	IsVariable   bool
	FunctionName string
	// if Condition
	IsCondition bool
	Condition   string
	IfBody      []Body
	ElseBody    []Body
	// Deklaration und/oder Initialisierung von Variablen
	IsDeclaration bool
	Variable      Attribute
}

type Method struct {
	AccessModifier string
	Name           string
	IsStatic       bool
	ReturnType     string
	Parameters     []Attribute
	MethodBody     []Body
	ReturnValue    string
}

type Interface struct {
	InterfaceName      string
	Inherits           []string
	AbstractAttributes []Attribute
	AbstractMethods    []Method
}

type ConstructorArg struct {
	Type  string
	Value string
}
