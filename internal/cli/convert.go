/*
package cli

import (

	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/MarmaidTranspiler/Merfolk/internal/reader"

)

	func Convert(args []string) {
		if len(args) < 2 {
			fmt.Println("specify both input and output directory")
			return
		}

		inputDir := args[0]
		outputDir := args[1]

		_, err := os.Stat(inputDir)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		_, err = os.Stat(outputDir)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		mdMatches, err := fs.Glob(os.DirFS(inputDir), "*.md")

		if err != nil {
			fmt.Println(err.Error())
			return
		}

		var files []string
		for _, match := range mdMatches {
			files = append(files, path.Join(inputDir, match))
		}

		var diagrams []reader.Diagram
		for _, file := range files {
			localDiagrams, err := reader.ParseFile(file)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			diagrams = append(diagrams, localDiagrams...)
		}

		// further conversion code here
		// this printing is just supposed to show parsing works
		for _, diagram := range diagrams {
			marshaled, err := json.MarshalIndent(diagram, "", "   ")
			if err != nil {
				panic("oh no")
			}
			fmt.Println(string(marshaled))
		}
	}
*/
package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/MarmaidTranspiler/Merfolk/internal/connector"
	"github.com/MarmaidTranspiler/Merfolk/internal/reader"
)

func Convert(args []string) {
	// Check input arguments
	if len(args) < 2 {
		fmt.Println("Specify both input and output directory.")
		return
	}

	inputDir := args[0]
	outputDir := args[1]

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

		// Process each diagram
		for _, diagram := range diagrams {
			if diagram.IsClass && diagram.Class != nil {
				// Use the connector to process the class diagram
				err := connector.TransformClassDiagram(
					diagram.Class,
					"templates/CompleteClassTemplate.tmpl", // Path to class template
					"templates/InterfaceTemplate.tmpl",     // Path to interface template
					outputDir,                              // Output directory
				)
				if err != nil {
					fmt.Println("Error processing class diagram in file", file, ":", err)
				} else {
					fmt.Println("Successfully processed class diagram from file:", file)
				}
			} else if diagram.IsSequence {
				// Sequence diagrams are not yet supported
				fmt.Println("Skipping sequence diagram in file:", file)
			} else {
				fmt.Println("Unknown diagram type in file:", file)
			}
		}
	}
}
