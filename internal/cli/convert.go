package cli

import (
	"fmt"
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

	// Parse Mermaid files and process diagrams
	for _, file := range files {
		fmt.Println("Processing file:", file)

		// Parse file into diagrams
		diagrams, err := reader.ParseFile(file)
		if err != nil {
			fmt.Println("Error parsing file", file, ":", err)
			continue
		}

		// Define template paths
		classTemplatePath := "internal/CodeTemplateGenerator/Templates/ClassTemplate.tmpl"
		interfaceTemplatePath := "internal/CodeTemplateGenerator/Templates/InterfaceTemplate.tmpl"

		//fmt.Println("Class Template Path:", classTemplatePath)
		//fmt.Println("Interface Template Path:", interfaceTemplatePath)
		//fmt.Println("Output Directory:", outputDir)

		// Process each diagram
		for _, diagram := range diagrams {
			if diagram.IsClass && diagram.Class != nil {
				// Use the connector to process the class diagram
				err := connector.TransformClassDiagram(
					diagram.Class,
					classTemplatePath,
					interfaceTemplatePath,
					outputDir,
				)
				if err != nil {
					fmt.Println("Error processing class diagram in file", file, ":", err)
				} else {
					fmt.Println("Successfully processed class diagram from file:", file)
				}
			} else if diagram.IsSequence && diagram.Sequence != nil {
				// Use the connector to process the sequence diagram
				err := connector.TransformSequenceDiagram(
					diagram.Sequence,
					classTemplatePath,
					outputDir,
				)
				if err != nil {
					fmt.Println("Error processing sequence diagram in file", file, ":", err)
				} else {
					fmt.Println("Successfully processed sequence diagram from file:", file)
				}
			} else {
				fmt.Println("Unknown or unsupported diagram type in file:", file)
			}
		}
	}
}
