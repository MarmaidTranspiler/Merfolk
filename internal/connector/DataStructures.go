package connector

type JavaClass struct {
	ClassName   string
	Abstraction string   // "abstract" or empty
	Inherits    string   // Superclass name
	Implements  []string // Interfaces implemented
	Attributes  []Attribute
	Methods     []Method
}

type InterfaceClass struct {
	InterfaceName string
	Inherits      []string // Super interfaces
	Methods       []Method
}

type Attribute struct {
	AccessModifier string
	Name           string
	Type           string
	ClassVariable  bool
	Constant       bool
	Value          interface{}
}

type Method struct {
	AccessModifier string
	Name           string
	Type           string
	Parameters     []Attribute
	Body           string
}
