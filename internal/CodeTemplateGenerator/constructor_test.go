package CodeTemplateGenerator

import (
	"bytes"
	"testing"
	"text/template"
)

func TestGenerateConstructor(t *testing.T) {
	testClass := Class{
		ClassName: "Person",
		Attributes: []Attribute{
			{
				Name:           "name",
				Type:           "String",
				AccessModifier: "private",
				IsObject:       true,
			},
			{
				Name:           "age",
				Type:           "int",
				AccessModifier: "private",
				IsObject:       false,
			},
			{
				Name:           "address",
				Type:           "Address",
				AccessModifier: "private",
				IsObject:       true,
				ObjectConstructorArgs: []ConstructorArg{
					{Type: "String", Value: "Musterstraße"},
					{Type: "int", Value: "42"},
				},
			},
			{
				Name:           "CONTACT_LIMIT",
				Type:           "int",
				AccessModifier: "public",
				IsConstant:     true,
			},
			{
				Name:            "staticCounter",
				Type:            "int",
				AccessModifier:  "private",
				IsClassVariable: true,
			},
		},
	}

	constructorTemplate := `public {{.ClassName}}() {
{{- range .Attributes }}
{{- if and (not .IsClassVariable) (not .IsConstant) (not .IsAttributeInitialized) }}
    this.{{.Name}} = {{ if .IsObject }}{{- if eq .Type "String" }}{{defaultZero .Type}}{{- else if .ObjectConstructorArgs }}new {{.Type}}({{range $index, $arg := .ObjectConstructorArgs}}{{if $index}}, {{end}}{{stringFormation $arg.Type $arg.Value}}{{end}}){{- else }}new {{.Type}}(){{- end }}{{- else }}{{defaultZero .Type}}
{{- end }};
{{- end }}
{{- end }}
}`

	tmpl, err := template.New("constructor").Funcs(TemplateGeneratorUtility()).Parse(constructorTemplate)
	if err != nil {
		t.Fatalf("Fehler beim Parsen des Templates: %v", err)
	}

	var output bytes.Buffer

	err = tmpl.Execute(&output, testClass)
	if err != nil {
		t.Fatalf("Fehler beim Ausführen des Templates: %v", err)
	}

	t.Logf("Generierter Konstruktor:\n%s", output.String())
}
