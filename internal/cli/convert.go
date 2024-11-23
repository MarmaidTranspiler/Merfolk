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

	if len(mdMatches) == 0 {
		fmt.Println("couldn't find markdown files in the input directory")
		return
	}

	var files []string
	for _, match := range mdMatches {
		files = append(files, path.Join(inputDir, match))
	}

	var diagrams []reader.Diagram
	for _, file := range files {
		localDiagrams, err := reader.ReadFile(file)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		diagrams = append(diagrams, localDiagrams...)
	}

	if len(diagrams) == 0 {
		fmt.Println("couldn't find mermaid diagrams in your markdown files")
		return
	}

	// TODO: further conversion code here
	// ! this printing is just supposed to show parsing works
	for _, diagram := range diagrams {
		marshaled, err := json.MarshalIndent(diagram, "", "   ")
		if err != nil {
			panic("oh no")
		}
		fmt.Println(string(marshaled))
	}
}
