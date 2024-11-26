package CodeTemplateGenerator

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"text/template"
)

func renderTemplate(tmplStr string, data any) (string, error) {

	tmpl, err := template.New("test").Funcs(TemplateGeneratorUtility()).Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func normalizeCode(input string) string {
	// Ersetzt alle Arten von Whitespaces (Leerzeichen, Tabs, Zeilenumbrüche) durch ein einziges Leerzeichen.
	re := regexp.MustCompile(`\s+`)
	return re.ReplaceAllString(strings.TrimSpace(input), " ")
}

func TestClassAttributes(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{AccessModifier: "private", Name: "id", Type: "int", IsClassVariable: false, IsConstant: false},
		},
	}
	expected := "private int id;"
	output, err := renderTemplate("{{range .Attributes}}{{.AccessModifier}} {{.Type}} {{.Name}};{{end}}", class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if output != expected {
		t.Errorf("Expected: %s, Got: %s", expected, output)
	}
}

func TestStaticClassAttributes(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{AccessModifier: "private", Name: "counter", Type: "int", IsClassVariable: true},
		},
	}
	expected := "private static int counter;"
	output, err := renderTemplate("{{range .Attributes}}{{if .IsClassVariable}}{{.AccessModifier}} static {{.Type}} {{.Name}};{{end}}{{end}}", class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if output != expected {
		t.Errorf("Expected: %s, Got: %s", expected, output)
	}
}

func TestConstantAttributes(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{AccessModifier: "public", Name: "PI", Type: "double", IsConstant: true, Value: 3.14},
		},
	}
	expected := "public final double PI = 3.14;"
	output, err := renderTemplate("{{range .Attributes}}{{if .IsConstant}}{{.AccessModifier}} final {{.Type}} {{.Name}} = {{.Value}};{{end}}{{end}}", class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected:\n%s\nGot:\n%s", normalizeCode(expected), normalizeCode(output))
	}
}

func TestConstructorWithInstanceAttributes(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{AccessModifier: "private", Name: "id", Type: "int"},
			{AccessModifier: "private", Name: "name", Type: "String"},
		},
	}
	expected :=
		`public TestClass(int id, String name) {
        	this.id = id;
        	this.name = name;
        }`
	output, err := renderTemplate(`
public {{.ClassName}}({{range $index, $attr := .Attributes}}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{end}}) {
	{{range .Attributes}}this.{{.Name}} = {{.Name}};
	{{end}}
}`, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected:\n%s\nGot:\n%s", normalizeCode(expected), normalizeCode(output))
	}
}

func TestGettersAndSetters(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{AccessModifier: "private", Name: "id", Type: "int"},
		},
	}
	expected := `
public int getId() { return this.id; }
public void setId(int id) { this.id = id; }`
	output, err := renderTemplate(`
{{range .Attributes}}
public {{.Type}} get{{title .Name}}() { return this.{{.Name}}; }
public void set{{title .Name}}({{.Type}} {{.Name}}) { this.{{.Name}} = {{.Name}}; }
{{end}}`, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected:\n%s\nGot:\n%s", normalizeCode(expected), normalizeCode(output))
	}
}

func TestMethodWithFunctionCallAndAssignment(t *testing.T) {
	method := Method{
		AccessModifier: "public",
		Name:           "calculate",
		ReturnType:     "int",
		Parameters: []Attribute{
			{Name: "a", Type: "int"},
			{Name: "b", Type: "int"},
		},
		MethodBody: []Body{
			{
				IsObjectCreation: false,
				FunctionName:     "sum",
				ObjFuncParameters: []Attribute{
					{Name: "a"},
					{Name: "b"},
				},
			},
		},
		ReturnValue: "result",
	}
	expected := `public int calculate(int a, int b) { 
	int result = sum(a, b); 
	return result;
}`
	output, err := renderTemplate(`
        {{.AccessModifier}} {{if .IsStatic}}static {{end}}{{.ReturnType}} {{.Name}}({{- range $index, $param := .Parameters }}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{- end }}) {
        {{ $method := . }}
        {{- range .MethodBody }}
        {{- if .IsObjectCreation }}
        {{ .ObjectType }} {{ .ObjectName }} = new {{ .ObjectType }} ({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- else }}
        {{- if $method.ReturnValue }}
        {{ $method.ReturnType }} {{ $method.ReturnValue }} = {{ end }}{{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});{{- end }}
        {{- end }}

        {{ if ne $method.ReturnType "void" }}return {{- if $method.ReturnValue }} {{ $method.ReturnValue }}{{ else }}{{ defaultZero $method.ReturnType }}{{ end }};{{- end }}
    }`, method)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected:\n%s\nGot:\n%s", normalizeCode(expected), normalizeCode(output))
	}
}

func TestVoidMethodWithObjectCreation(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "createObject",
				ReturnType:     "void",
				MethodBody: []Body{
					{
						IsObjectCreation:  true,
						ObjectName:        "myObject",
						ObjectType:        "MyObject",
						ObjFuncParameters: []Attribute{},
					},
				},
			},
		},
	}

	expected := `
	public void createObject() {
	    MyObject myObject = new MyObject();
	}
	`

	output, err := renderTemplate("{{range .Methods}}{{.AccessModifier}} {{.ReturnType}} {{.Name}}() {\n{{range .MethodBody}}{{if .IsObjectCreation}}{{.ObjectType}} {{.ObjectName}} = new {{.ObjectType}}();\n{{end}}{{end}}}\n{{end}}", class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestVoidMethodWithObjectCreationAndParams(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "createObjectWithParams",
				ReturnType:     "void",
				MethodBody: []Body{
					{
						IsObjectCreation: true,
						ObjectName:       "myObject",
						ObjectType:       "MyObject",
						ObjFuncParameters: []Attribute{
							{Name: "param1", Type: "int", Value: 42},
							{Name: "param2", Type: "String", Value: `"Hello"`},
						},
					},
				},
			},
		},
	}

	expected := `
	public void createObjectWithParams() {
	    MyObject myObject = new MyObject(42, "Hello");
	}
	`

	output, err := renderTemplate("{{range .Methods}}{{.AccessModifier}} {{.ReturnType}} {{.Name}}() {\n{{range .MethodBody}}{{if .IsObjectCreation}}{{.ObjectType}} {{.ObjectName}} = new {{.ObjectType}}({{range $index, $param := .ObjFuncParameters}}{{if $index}}, {{end}}{{if $param.Value}}{{$param.Value}}{{else}}{{$param.Name}}{{end}}{{end}});\n{{end}}{{end}}}\n{{end}}", class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestVoidMethodWithFunctionCall(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "callFunction",
				ReturnType:     "void",
				MethodBody: []Body{
					{
						FunctionName: "someFunction",
						ObjFuncParameters: []Attribute{
							{Name: "param1", Type: "int", Value: 42},
						},
					},
				},
			},
		},
	}

	expected := `
	public void callFunction() {
	    someFunction(42);
	}
	`

	output, err := renderTemplate("{{range .Methods}}{{.AccessModifier}} {{.ReturnType}} {{.Name}}() {\n{{range .MethodBody}}{{if .FunctionName}}{{.FunctionName}}({{range $index, $param := .ObjFuncParameters}}{{if $index}}, {{end}}{{if $param.Value}}{{$param.Value}}{{else}}{{$param.Name}}{{end}}{{end}});\n{{end}}{{end}}}\n{{end}}", class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestSimpleInterface(t *testing.T) {
	Tinterface := Interface{
		InterfaceName: "SimpleInterface",
		AbstractMethods: []Method{
			{
				Name:       "doSomething",
				ReturnType: "void",
				Parameters: []Attribute{
					{Name: "param1", Type: "int"},
				},
			},
		},
	}

	expected := `
	public interface SimpleInterface {
	    void doSomething(int param1);
	}
	`

	output, err := renderTemplate(`public interface {{.InterfaceName}} {
	{{range .AbstractMethods}}
	    {{.ReturnType}} {{.Name}}({{range $index, $param := .Parameters}}{{if $index}}, {{end}}{{$param.Type}} {{$param.Name}}{{end}});
	{{end}}
	}`, Tinterface)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestInterfaceWithMultipleInheritance(t *testing.T) {
	Interface := Interface{
		InterfaceName: "AdvancedInterface",
		Inherits:      []string{"BaseInterface", "AnotherInterface"},
		AbstractMethods: []Method{
			{
				Name:       "doAdvanced",
				ReturnType: "String",
				Parameters: []Attribute{},
			},
		},
	}

	expected := `
	public interface AdvancedInterface extends BaseInterface, AnotherInterface {
	   public String doAdvanced();
	}
	`

	output, err := renderTemplate(`public interface {{.InterfaceName}} {{- if .Inherits}} extends {{ range $index, $interface := .Inherits}}{{if $index}}, {{end}}{{$interface}}{{- end}} {{- end}} {
{{- range .AbstractAttributes }}
    public {{.Type}} {{.Name}}{{if .Value}} = {{stringFormation .Type .Value}}{{- end}};
{{- end }}

{{ range .AbstractMethods }}
    public {{.ReturnType}} {{.Name}}();
{{- end }}
}`, Interface)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestInterfaceWithAbstractAttributes(t *testing.T) {
	Interface := Interface{
		InterfaceName: "AttributeInterface",
		AbstractAttributes: []Attribute{
			{AccessModifier: "public", Name: "CONST_VALUE", Type: "int", IsConstant: true, Value: 100},
			{AccessModifier: "public", Name: "ID", Type: "String", IsConstant: true, Value: nil},
		},
	}

	expected := `
	public interface AttributeInterface {
	    public static final int CONST_VALUE = 100;
	    public static final String ID;
	}
	`

	output, err := renderTemplate(`public interface {{.InterfaceName}} {
	{{range .AbstractAttributes}}
	    {{.AccessModifier}} static final {{.Type}} {{.Name}}{{if .Value}} = {{.Value}}{{end}};
	{{end}}
	}`, Interface)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestClassExtendsSuperclass(t *testing.T) {
	class := Class{
		ClassName: "ChildClass",
		Inherits:  "BaseClass",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "doSomething",
				ReturnType:     "void",
				Parameters:     []Attribute{},
			},
		},
	}

	expected := `
	public class ChildClass extends BaseClass {
	    public void doSomething() {
	    }
	}
	`

	output, err := renderTemplate(`public class {{.ClassName}} {{if .Inherits}}extends {{.Inherits}} {{end}}{
	{{range .Methods}}
	    {{.AccessModifier}} {{.ReturnType}} {{.Name}}({{range $index, $param := .Parameters}}{{if $index}}, {{end}}{{$param.Type}} {{$param.Name}}{{end}}) {
	    }
	{{end}}
	}`, class)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestClassImplementsInterface(t *testing.T) {
	class := Class{
		ClassName: "ImplementingClass",
		Abstraction: []string{
			"SomeInterface",
		},
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "performAction",
				ReturnType:     "String",
				Parameters: []Attribute{
					{Name: "input", Type: "String"},
				},
			},
		},
	}

	expected := `
	public class ImplementingClass implements SomeInterface {
	    public String performAction(String input) {
		return "";
		}
	}
	`

	output, err := renderTemplate(`public class {{.ClassName}}{{if .Inherits}} extends {{.Inherits}}{{end}}{{if gt (len .Abstraction) 0}} implements {{ range $index, $item := .Abstraction}}{{if $index}}, {{end}}{{$item}}{{- end}}{{end}} {
    {{- range .Attributes }}
    {{.AccessModifier}}{{if .IsClassVariable}} static{{end}}{{if .IsConstant}} final{{end}} {{.Type}} {{.Name}} {{- if .Value}} = {{stringFormation .Type .Value}}{{- end}};
    {{- end }}
	{{- range .Methods }}
    {{.AccessModifier}} {{if .IsStatic}}static {{end}}{{.ReturnType}} {{.Name}}({{- range $index, $param := .Parameters }}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{- end }}) {
        {{ $method := . }}
        {{- range .MethodBody }}
        {{- if .IsObjectCreation }}
        {{ .ObjectType }} {{ .ObjectName }} = new {{ .ObjectType }} ({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- else }}
        {{- if $method.ReturnValue }}
        {{ $method.ReturnType }} {{ $method.ReturnValue }} = {{ end }}{{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});{{- end }}
        {{- end }}

        {{ if ne $method.ReturnType "void" }}return {{- if $method.ReturnValue }} {{ $method.ReturnValue }}{{ else }} {{ defaultZero $method.ReturnType }}{{ end }};{{- end }}
    }
{{- end }}

	}`, class)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestClassExtendsAndImplements(t *testing.T) {
	class := Class{
		ClassName: "ComplexClass",
		Inherits:  "BaseClass",
		Abstraction: []string{
			"InterfaceA",
			"InterfaceB",
		},
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "complexMethod",
				ReturnType:     "int",
				Parameters: []Attribute{
					{Name: "value", Type: "int"},
				},
			},
		},
	}

	expected := `
	public class ComplexClass extends BaseClass implements InterfaceA, InterfaceB {
	    public int complexMethod(int value) {
			return 0;
	    }
	}
	`

	output, err := renderTemplate(`public class {{.ClassName}}{{if .Inherits}} extends {{.Inherits}}{{end}}{{if gt (len .Abstraction) 0}} implements {{ range $index, $item := .Abstraction}}{{if $index}}, {{end}}{{$item}}{{- end}}{{end}} {
    {{- range .Attributes }}
    {{.AccessModifier}}{{if .IsClassVariable}} static{{end}}{{if .IsConstant}} final{{end}} {{.Type}} {{.Name}} {{- if .Value}} = {{stringFormation .Type .Value}}{{- end}};
    {{- end }}
	{{- range .Methods }}
    {{.AccessModifier}} {{if .IsStatic}}static {{end}}{{.ReturnType}} {{.Name}}({{- range $index, $param := .Parameters }}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{- end }}) {
        {{ $method := . }}
        {{- range .MethodBody }}
        {{- if .IsObjectCreation }}
        {{ .ObjectType }} {{ .ObjectName }} = new {{ .ObjectType }} ({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- else }}
        {{- if $method.ReturnValue }}
        {{ $method.ReturnType }} {{ $method.ReturnValue }} = {{ end }}{{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});{{- end }}
        {{- end }}

        {{ if ne $method.ReturnType "void" }}return {{- if $method.ReturnValue }} {{ $method.ReturnValue }}{{ else }} {{ defaultZero $method.ReturnType }}{{ end }};{{- end }}
    }
{{- end }}

	}`, class)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestMethodReturnsObject(t *testing.T) {
	class := Class{
		ClassName: "ObjectReturningClass",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "createObject",
				ReturnType:     "CustomObject",
				MethodBody: []Body{
					{
						IsObjectCreation: true,
						ObjectName:       "obj",
						ObjectType:       "CustomObject",
						ObjFuncParameters: []Attribute{
							{Name: "value", Type: "String", Value: "Test"},
						},
					},
				},
				ReturnValue: "obj",
			},
		},
	}

	expected := `
	    public class ObjectReturningClass {
	    public CustomObject createObject() {
	        CustomObject obj = new CustomObject("Test");
	        return obj;
	    }
	}`

	output, err := renderTemplate(`public class {{.ClassName}}{{if .Inherits}} extends {{.Inherits}}{{end}}{{if gt (len .Abstraction) 0}} implements {{- range $index, $item := .Abstraction}}{{if $index}}, {{end}}{{$item}}{{- end}}{{end}} {
{{- range .Methods }}
    {{.AccessModifier}} {{if .IsStatic}}static {{end}}{{.ReturnType}} {{.Name}}({{- range $index, $param := .Parameters }}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{- end }}) {
        {{ $method := . }}
        {{- range .MethodBody }}
        {{- if .IsObjectCreation }}
        {{ .ObjectType }} {{ .ObjectName }} = new {{ .ObjectType }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ stringFormation $param.Type $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- else }}
        {{- if $method.ReturnValue }}
        {{ $method.ReturnType }} {{ $method.ReturnValue }} = {{ end }}{{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});{{- end }}
        {{- end }}

        {{ if ne $method.ReturnType "void" }}return {{- if $method.ReturnValue }} {{ $method.ReturnValue }}{{ else }} {{ defaultZero $method.ReturnType }}{{ end }};{{- end }}
    }
{{- end }}
}`, class)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestMethodWithParametersReturnsObject(t *testing.T) {
	class := Class{
		ClassName: "ParameterizedObjectCreator",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "createCustomObject",
				ReturnType:     "CustomObject",
				Parameters: []Attribute{
					{Name: "name", Type: "String"},
					{Name: "age", Type: "int"},
				},
				MethodBody: []Body{
					{
						IsObjectCreation: true,
						ObjectName:       "obj",
						ObjectType:       "CustomObject",
						ObjFuncParameters: []Attribute{
							{Name: "name", Type: "String"},
							{Name: "age", Type: "int"},
						},
					},
				},
				ReturnValue: "obj",
			},
		},
	}

	expected := `
	public class ParameterizedObjectCreator {
	    public CustomObject createCustomObject(String name, int age) {
	        CustomObject obj = new CustomObject(name, age);
	        return obj;
	    }
	}
	`

	output, err := renderTemplate(`public class {{.ClassName}}{{if .Inherits}} extends {{.Inherits}}{{end}}{{if gt (len .Abstraction) 0}} implements {{- range $index, $item := .Abstraction}}{{if $index}}, {{end}}{{$item}}{{- end}}{{end}} {
	{{- range .Methods }}
    {{.AccessModifier}} {{if .IsStatic}}static {{end}}{{.ReturnType}} {{.Name}}({{- range $index, $param := .Parameters }}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{- end }}) {
        {{ $method := . }}
        {{- range .MethodBody }}
        {{- if .IsObjectCreation }}
        {{ .ObjectType }} {{ .ObjectName }} = new {{ .ObjectType }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ stringFormation $param.Typ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- else }}
        {{- if $method.ReturnValue }}
        {{ $method.ReturnType }} {{ $method.ReturnValue }} = {{ end }}{{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});{{- end }}
        {{- end }}

        {{ if ne $method.ReturnType "void" }}return {{- if $method.ReturnValue }} {{ $method.ReturnValue }}{{ else }} {{ defaultZero $method.ReturnType }}{{ end }};{{- end }}
    }
{{- end }}
}`, class)

	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}
