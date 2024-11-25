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
	AccessModifier  string
	Name            string
	Type            string
	IsClassVariable bool
	IsConstant      bool
	Value           any
}

type Method struct {
	AccessModifier string
	Name           string
	IsStatic       bool
	Type           string
	Parameters     []Attribute
	Body           []string
}

type Interface struct {
	InterfaceName      string
	Inherits           []string
	AbstractAttributes []Attribute
	AbstractMethods    []Method
}
