package cli

import (
	"fmt"
	generator "github.com/MarmaidTranspiler/Merfolk/internal/CodeTemplateGenerator"
	"github.com/MarmaidTranspiler/Merfolk/internal/connector"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
	"io/fs"
	"os"
	"path"
)

func Convert(args []string) {
	// Check input arguments
	if len(args) < 2 {
		fmt.Println("Specify both input and output directory.")
		return
	}

	inputDir := args[0]
	outputDir := args[1]

	// Print the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error fetching current working directory:", err)
		return
	}
	fmt.Println("Current Working Directory:", cwd)

	// Check input directory
	if _, err := os.Stat(inputDir); os.IsNotExist(err) {
		fmt.Println("Input directory does not exist:", inputDir)
		return
	}

	// Ensure output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			fmt.Println("Failed to create output directory:", err)
			return
		}
	}

	// Find all .md files in the input directory
	mdMatches, err := fs.Glob(os.DirFS(inputDir), "*.md")
	if err != nil {
		fmt.Println("Error reading input directory:", err)
		return
	}

	var files []string
	for _, match := range mdMatches {
		files = append(files, path.Join(inputDir, match))
	}

	// Initialize maps for classes and interfaces
	classes := make(map[string]*generator.Class)
	interfaces := make(map[string]*generator.Interface)
	var sequenceDiagrams []*reader.SequenceDiagram

	// Parse Mermaid files
	for _, file := range files {
		fmt.Println("Processing file:", file)

		// Parse file into diagrams
		diagrams, err := reader.ParseFile(file)
		if err != nil {
			fmt.Println("Error parsing file", file, ":", err)
			continue
		}

		// Separate class and sequence diagrams
		for _, diagram := range diagrams {
			if diagram.IsClass && diagram.Class != nil {
				// Process class diagrams
				classMap, interfaceMap, err := connector.TransformClassDiagram(
					diagram.Class,
					"internal/CodeTemplateGenerator/Templates/ClassTemplate.tmpl",     // No immediate output yet
					"internal/CodeTemplateGenerator/Templates/InterfaceTemplate.tmpl", // No immediate output yet
					outputDir+"/",
				)
				if err != nil {
					fmt.Println("Error processing class diagram in file", file, ":", err)
					continue
				}

				// Merge class and interface maps
				for k, v := range classMap {
					classes[k] = v
				}
				for k, v := range interfaceMap {
					interfaces[k] = v
				}
			} else if diagram.IsSequence && diagram.Sequence != nil {
				// Store sequence diagrams for later processing
				sequenceDiagrams = append(sequenceDiagrams, diagram.Sequence)
			} else {
				fmt.Println("Unknown or unsupported diagram type in file:", file)
			}
		}
	}

	// Process sequence diagrams and integrate with classes
	for _, sequenceDiagram := range sequenceDiagrams {
		/*classes, err = */ connector.TransformSequenceDiagram(
			sequenceDiagram,
			classes, // Modify existing class definitions
			"internal/CodeTemplateGenerator/Templates/ClassTemplate.tmpl", // No immediate output yet
			outputDir+"/",
		)
		if err != nil {
			fmt.Println("Error processing sequence diagram:", err)
		} else {
			fmt.Println("Successfully processed sequence diagram.")
		}
	}

	// Generate Java code for all classes and interfaces
	classTemplatePath := "internal/CodeTemplateGenerator/Templates/ClassTemplate.tmpl"
	interfaceTemplatePath := "internal/CodeTemplateGenerator/Templates/InterfaceTemplate.tmpl"

	for _, class := range classes {
		err := generator.GenerateJavaCode(*class, outputDir+"/", class.ClassName, classTemplatePath)
		if err != nil {
			fmt.Println("Error generating Java class:", class.ClassName, ":", err)
		}
	}

	for _, iface := range interfaces {
		err := generator.GenerateJavaCode(*iface, outputDir+"/", iface.InterfaceName, interfaceTemplatePath)
		if err != nil {
			fmt.Println("Error generating Java interface:", iface.InterfaceName, ":", err)
		}
	}
}
