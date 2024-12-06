package CodeTemplateGenerator

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"text/template"
)

var PATHCLASS = `
public class {{.ClassName}}{{if .Inherits}} extends {{.Inherits}}{{end}}{{if gt (len .Abstraction) 0}} implements {{ range $index, $item := .Abstraction}}{{if $index}}, {{end}}{{$item}}{{- end}}{{end}} {
    {{- range .Attributes }}
    {{ $attribute := . }}
    {{.AccessModifier}}{{if .IsClassVariable}} static{{end}}{{if .IsConstant}} final{{end}} {{.Type}} {{.Name}} {{- if .IsAttributeInitialized}} = {{- if .IsObject }} new {{.Type}}( {{- range $index, $arg := .ObjectConstructorArgs}} {{- if $index}}, {{end}}{{stringFormation $arg.Type $arg.Value}} {{- end}} ) {{- else}} {{stringFormation .Type .Value}}{{- end}}{{- end }};
    {{- end }}

    // default constructor
    public {{.ClassName}}() {
        {{- range .Attributes }}
        {{- if and (not .IsClassVariable) (not .IsConstant) (not .IsAttributeInitialized) }}
        this.{{.Name}} = {{ if .IsObject }}new {{.Type}}({{range $index, $arg := .ObjectConstructorArgs}}{{if $index}}, {{end}}{{stringFormation $arg.Type $arg.Value}}{{end}}){{ else }}{{defaultZero .Type}}{{- end }};
        {{- end}}
        {{- end }}
    }

    // constructor with all arguments
    public {{.ClassName}}(
    {{- range $index, $field := .Attributes }}
        {{- if and (not $field.IsClassVariable) (not $field.IsConstant) }}{{- if $index }}, {{ end }}{{$field.Type}} {{$field.Name}} {{- end }}{{- end }}) {
        {{- range .Attributes }}
        {{- if and (not .IsClassVariable) (not .IsConstant) }}
        this.{{.Name}} = {{.Name}};
        {{- end}}
        {{- end }}
    }

    {{- range .Attributes }}
    {{- if and (ne .AccessModifier "public") (not .IsClassVariable) (not .IsConstant) }}
    // Getter {{.Name}}
    public {{.Type}} get{{title .Name}}() {
        return {{.Name}};
    }
    // Setter {{.Name}}
    public void set{{title .Name}}({{.Type}} {{.Name}}) {
        this.{{.Name}} = {{.Name}};
    }
    {{end}}
    {{- end }}

{{- range .Methods }}
    {{.AccessModifier}} {{if .IsStatic}}static {{end}}{{.ReturnType}} {{.Name}}({{- range $index, $param := .Parameters }}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{- end }}) {
        {{- $method := . }}

        {{- range .MethodBody }}
        {{- if .IsDeclaration }}
        {{- with .Variable }}
        {{ .Type }} {{ .Name }}{{ if .IsAttributeInitialized }} = {{ .Value }}{{ end }};
        {{- end }}
        {{- else if .IsCondition }}
        if ({{ .Condition }}) {
            {{- range .IfBody }}
            {{- template "BodyTemplate" . }}
            {{- end }}
        }
        {{- if .ElseBody }}
        else {
            {{- range .ElseBody }}
            {{- template "BodyTemplate" . }}
            {{- end }}
        }
        {{- end }}
        {{- else if .IsObjectCreation }}
        {{ .ObjectType }} {{ .ObjectName }} = new {{ .ObjectType }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ stringFormation $param.Typ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- else if .IsVariable }}
        {{ .Type }} {{ .Name }} = {{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- else }}
        {{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{- if $index }}, {{ end }}{{- if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
        {{- end }}
        

        {{- if ne $method.ReturnType "void" }}
        return {{ if $method.ReturnValue }}{{ if eq .IsVariable false }}{{ stringFormation $method.ReturnType $method.ReturnValue }}{{ else }}{{ $method.ReturnValue }}{{ end }}{{ else }}{{ defaultZero $method.ReturnType }}{{ end }};
        {{- end }}
		{{- end }}
    }
{{- end }}

}

{{- define "BodyTemplate" }}
{{- if .IsDeclaration }}
{{- range .Variable }}
{{ .Type }} {{ .Name }}{{ if .Value }} = {{ stringFormation .Type .Value }}{{ end }};
{{- end }}
{{- else if .IsObjectCreation }}
{{ .ObjectType }} {{ .ObjectName }} = new {{ .ObjectType }}({{- range $index, $param := .ObjFuncParameters }}{{ if $index }}, {{ end }}{{ if $param.Value }}{{ stringFormation $param.Type $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
{{- else if .IsCondition }}
if ({{ .Condition }}) {
    {{- range .IfBody }}
    {{- template "BodyTemplate" . }}
    {{- end }}
}
{{- if .ElseBody }}
else {
    {{- range .ElseBody }}
    {{- template "BodyTemplate" . }}
    {{- end }}
}
{{- end }}
{{- else }}
{{ .FunctionName }}({{- range $index, $param := .ObjFuncParameters }}{{ if $index }}, {{ end }}{{ if $param.Value }}{{ $param.Value }}{{ else }}{{ $param.Name }}{{ end }}{{- end }});
{{- end }}
{{- end }}`

var PATHINTERFACE = `
public interface {{.InterfaceName}} {{- if .Inherits}} extends {{ range $index, $interface := .Inherits}}{{if $index}}, {{end}}{{$interface}}{{- end}} {{- end}} {
{{- range .AbstractAttributes }}
    public {{if .IsClassVariable}} static{{end}}{{if .IsConstant}} final{{end}} {{.Type}} {{.Name}} {{- if .IsAttributeInitialized}} = {{- if .IsObject }} new {{.Type}}( {{- range $index, $arg := .ObjectConstructorArgs}} {{- if $index}}, {{end}}{{stringFormation $arg.Type $arg.Value}} {{- end}} ) {{- else}} {{stringFormation .Type .Value}}{{- end}}{{- end }};
{{- end }}

{{ range .AbstractMethods }}
    public {{.ReturnType}} {{.Name}}({{- range $index, $param := .Parameters }}{{if $index}}, {{end}}{{.Type}} {{.Name}}{{- end }});
{{- end }}
}`

func renderTemplate(tmplStr string, data any) (string, error) {

	templateStruct, err := template.New("test").Funcs(TemplateGeneratorUtility()).Parse(tmplStr)
	//templateStruct, err := template.New("test").Funcs(TemplateGeneratorUtility()).ParseFiles(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := templateStruct.Execute(&buf, data); err != nil {
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
	expected := `
        public class TestClass {
            
            private int id;
        
            // default constructor
            public TestClass() {
                this.id = 0;
            }
        
            // constructor with all arguments
            public TestClass(int id) {
                this.id = id;
            }
            // Getter id
            public int getId() {
                return id;
            }
            // Setter id
            public void setId(int id) {
                this.id = id;
            }
        }`
	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected: %s, Got: %s", expected, output)
	}
}

func TestStaticClassAttributes(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{
				AccessModifier:  "private",
				Name:            "counter",
				Type:            "int",
				IsClassVariable: true,
			},
		},
	}

	expected := `
    public class TestClass { 
		private static int counter; 

	// default constructor 
	public TestClass() {
	} 

	// constructor with all arguments 
	public TestClass() { 
	} 
}`

	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected: %s, Got: %s", normalizeCode(expected), normalizeCode(output))
	}
}

func TestConstantAttributes(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{
				AccessModifier:         "public",
				Name:                   "PI",
				Type:                   "double",
				IsConstant:             true,
				IsAttributeInitialized: true,
				Value:                  3.14,
			},
		},
	}

	expected := `
    public class TestClass { 
		public final double PI = 3.14; 
	
		// default constructor 
		public TestClass() {
		
		} 
		// constructor with all arguments 
		public TestClass() { 
		
		} 
	}`

	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected: %s, Got: %s", normalizeCode(expected), normalizeCode(output))
	}
}

func TestConstructorWithInstanceAttributes(t *testing.T) {
	class := Class{
		ClassName: "TestClass",
		Attributes: []Attribute{
			{
				AccessModifier: "private",
				Name:           "id",
				Type:           "int",
			},
			{
				AccessModifier: "private",
				Name:           "name",
				Type:           "String",
			},
		},
	}

	expected := `
	public class TestClass {
        private int id;
        private String name;
	
		// default constructor 
		public TestClass() { 
			this.id = 0; 
			this.name = ""; 
		} 
		// constructor with all arguments 
		public TestClass(int id, String name) {
			this.id = id; 
			this.name = name; 
		} 
		// Getter id 
		public int getId() { 
			return id; 
		} 
		// Setter id 
		public void setId(int id) { 
			this.id = id; 
		} 
		// Getter name 
		public String getName() {
			return name; 
		} 
		// Setter name 
		public void setName(String name) {
			this.name = name; 
		} 
	}`

	output, err := renderTemplate(PATHCLASS, class)
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
	public class TestClass {
		private int id; 

		// default constructor 
		public TestClass() { 
			this.id = 0; 
		} 
		
		// constructor with all arguments 
		public TestClass(int id) { 
			this.id = id; 
		} 
		
		// Getter id 
		public int getId() { 
			return id; 
		} 

		// Setter id 
		public void setId(int id) { 
			this.id = id; 
		} 
	}`

	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}
	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Expected:\n%s\nGot:\n%s", normalizeCode(expected), normalizeCode(output))
	}
}

func TestMethodWithFunctionCallAndAssignment(t *testing.T) {

	method := Class{
		ClassName: "TestClass",
		Methods: []Method{{
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
		}},
	}
	expected := `
	public class TestClass {
		// default constructor 
		public TestClass() { 
		} 
		// constructor with all arguments 
		public TestClass() { 
		} 
		public int calculate(int a, int b) { 
		sum(a, b); 
		return result; 
		} 
	}`

	output, err := renderTemplate(PATHCLASS, method)
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
	public class TestClass { 
		
		// default constructor 
		public TestClass() {
		} 
		// constructor with all arguments 
		public TestClass() {
		} 
		public void createObject() {
			MyObject myObject = new MyObject(); 
		} 
	}`

	output, err := renderTemplate(PATHCLASS, class)
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
							{Name: "param2", Type: "String", Value: "Hello"},
						},
					},
				},
			},
		},
	}

	expected := `
	public class TestClass { 
		
		// default constructor 
		public TestClass() {
		} 
		// constructor with all arguments 
		public TestClass() {
		} 
		public void createObjectWithParams() {
	    	MyObject myObject = new MyObject(42, "Hello");
		}
	}
	`

	output, err := renderTemplate(PATHCLASS, class)
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
	public class TestClass { 
		
		// default constructor 
		public TestClass() {
		} 
		// constructor with all arguments 
		public TestClass() {
		} 
		public void callFunction() {
	    	someFunction(42);
		}
	}`

	output, err := renderTemplate(PATHCLASS, class)
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
	    public void doSomething(int param1);
	}
	`

	output, err := renderTemplate(PATHINTERFACE, Tinterface)

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

	output, err := renderTemplate(PATHINTERFACE, Interface)

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
			{AccessModifier: "public", Name: "CONST_VALUE", Type: "int", IsConstant: true, IsClassVariable: true, Value: 100, IsAttributeInitialized: true},
			{AccessModifier: "public", Name: "ID", Type: "String", IsConstant: true, IsClassVariable: true},
		},
	}

	expected := `
	public interface AttributeInterface {
	    public static final int CONST_VALUE = 100;
	    public static final String ID;
	}
	`

	output, err := renderTemplate(PATHINTERFACE, Interface)

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
		// default constructor 
		public ChildClass() {
		} 
		// constructor with all arguments 
		public ChildClass() {
		} 
		public void doSomething() {
		} 
	}`

	output, err := renderTemplate(PATHCLASS, class)

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
		
		// default constructor 
		public ImplementingClass() {
		} 
		// constructor with all arguments 
		public ImplementingClass() {
		}
		public String performAction(String input) {
		return "";
		}
	}
	`

	output, err := renderTemplate(PATHCLASS, class)

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

	output, err := renderTemplate(PATHCLASS, class)

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
							{Type: "String", Value: "Test"},
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

	output, err := renderTemplate(PATHCLASS, class)

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
	
		// default constructor 
		public ParameterizedObjectCreator() {
		} 
		
		// constructor with all arguments 
		public ParameterizedObjectCreator() {
		} 
		public CustomObject createCustomObject(String name, int age) { 
			CustomObject obj = new CustomObject(name, age); 
			return obj; 
		} 
	}`

	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestMethodInteractionExample(t *testing.T) {
	class := Class{
		ClassName: "MethodInteractionExample",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "getNiceString",
				ReturnType:     "String",
				ReturnValue:    "nice",
				MethodBody:     []Body{},
			},
			{
				AccessModifier: "public",
				Name:           "printString",
				ReturnType:     "void",
				Parameters: []Attribute{
					{Name: "input", Type: "String"},
				},
				MethodBody: []Body{
					{
						FunctionName: "System.out.println",
						ObjFuncParameters: []Attribute{
							{Name: "input", Type: "String"},
						},
					},
				},
			},
			{
				AccessModifier: "public",
				Name:           "callMethods",
				ReturnType:     "void",
				MethodBody: []Body{
					{
						FunctionName:     "getNiceString",
						ObjectName:       "result",
						ObjectType:       "String",
						IsObjectCreation: false,
					},
					{
						FunctionName: "printString",
						ObjFuncParameters: []Attribute{
							{Name: "result", Type: "String"},
						},
					},
				},
			},
		},
	}

	expected := `
    public class MethodInteractionExample {
        // default constructor 
		public MethodInteractionExample() {
		} 
		// constructor with all arguments 
		public MethodInteractionExample() {
		} 

		public String getNiceString() {
            return "nice";
        }

        public void printString(String input) {
            System.out.println(input);
        }

        public void callMethods() {
            String result = getNiceString();
            printString(result);
        }
    }
    `

	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestComplexMethodWithConditions(t *testing.T) {
	class := Class{
		ClassName: "ComplexLogicExample",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "processValue",
				ReturnType:     "String",
				Parameters: []Attribute{
					{Name: "value", Type: "int"},
				},
				MethodBody: []Body{
					{
						IsDeclaration: true,
						ObjFuncParameters: []Attribute{
							{Name: "result", Type: "String", Value: "\"Default\"", IsAttributeInitialized: true},
						},
					},
					{
						IsCondition: true,
						Condition:   "value > 0",
						IfBody: []Body{
							{
								FunctionName: "System.out.println",
								ObjFuncParameters: []Attribute{
									{Value: "\"Positive value\"", Type: "String"},
								},
							},
							{
								FunctionName: "result.concat",
								ObjFuncParameters: []Attribute{
									{Value: " Processed\"", Type: "String"},
								},
							},
						},
						ElseBody: []Body{
							{
								FunctionName: "System.out.println",
								ObjFuncParameters: []Attribute{
									{Value: "\"Non-positive value\"", Type: "String"},
								},
							},
						},
					},
					{
						FunctionName: "return",
						ObjFuncParameters: []Attribute{
							{Value: "result", Type: "String"},
						},
					},
				},
			},
		},
	}

	expected := `
    public class ComplexLogicExample {
		
		// default constructor 
		public ComplexLogicExample() { 
		} 
		// constructor with all arguments 
		public ComplexLogicExample() {
		}
        public String processValue(int value) {
            String result = "Default";
            if (value > 0) {
                System.out.println("Positive value");
                result.concat(" Processed");
            } else {
                System.out.println("Non-positive value");
            }
            return result;
        }
    }
    `

	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}

func TestStaticMethodWithMultipleParameters(t *testing.T) {
	class := Class{
		ClassName: "MathUtils",
		Methods: []Method{
			{
				AccessModifier: "public",
				Name:           "calculateSum",
				IsStatic:       true,
				ReturnType:     "int",
				Parameters: []Attribute{
					{Name: "a", Type: "int"},
					{Name: "b", Type: "int"},
					{Name: "c", Type: "int"},
				},
				MethodBody: []Body{
					{
						IsDeclaration: true,
						ObjFuncParameters: []Attribute{
							{Name: "sum", Type: "int", Value: "a + b + c", IsAttributeInitialized: true},
						},
					},
					{
						IsCondition: true,
						Condition:   "sum > 0",
						IfBody: []Body{
							{
								FunctionName: "System.out.println",
								ObjFuncParameters: []Attribute{
									{Value: "\"Positive sum\"", Type: "String"},
								},
							},
						},
					},
					{
						FunctionName: "return",
						ObjFuncParameters: []Attribute{
							{Value: "sum", Type: "int"},
						},
					},
				},
			},
		},
	}

	expected := `
    public class MathUtils {
		
		// default constructor 
		public MathUtils() {
		} 
		// constructor with all arguments 
		public MathUtils() {
		}
        public static int calculateSum(int a, int b, int c) {
            int sum = a + b + c;
            if (sum > 0) {
                System.out.println("Positive sum");
            }
            return sum;
        }
    }
    `

	output, err := renderTemplate(PATHCLASS, class)
	if err != nil {
		t.Fatalf("Error rendering template: %v", err)
	}

	if normalizeCode(output) != normalizeCode(expected) {
		t.Errorf("Mismatch!\nExpected:\n%s\nGot:\n%s\n", normalizeCode(expected), normalizeCode(output))
	}
}
