package reader

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

type Diagram struct {
	IsSequence bool
	Sequence   *SequenceDiagram

	IsClass bool
	Class   *ClassDiagram
}

func ParseFile(dir string) ([]Diagram, error) {
	file, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var builder strings.Builder
	reading := false
	var diagrams []Diagram

	for scanner.Scan() {
		line := scanner.Text()

		if line == "```mermaid" {
			if reading {
				return nil, errors.New("nested diagram")
			}
			reading = true
			continue
		} else if line == "```" {
			if reading {
				reading = false
				diagram, err := ParseDiagram(builder.String())
				if err != nil {
					return nil, err
				}
				builder.Reset()
				diagrams = append(diagrams, *diagram)
			}
			continue
		}

		if reading {
			builder.WriteString(line + "\n")
		}
	}

	if reading {
		return nil, errors.New("unclosed diagram")
	}

	return diagrams, nil
}

func ParseDiagram(input string) (*Diagram, error) {
	var diagram Diagram

	// Trim leading and trailing whitespace
	trimmedInput := strings.TrimSpace(input)

	if strings.HasPrefix(trimmedInput, "classDiagram") {
		parsed, err := ClassDiagramParser.ParseString("", trimmedInput)
		if err != nil {
			return nil, err
		}
		diagram = Diagram{IsClass: true, Class: parsed}
	} else if strings.HasPrefix(trimmedInput, "sequenceDiagram") {
		parsed, err := SequenceDiagramParser.ParseString("", trimmedInput)
		if err != nil {
			return nil, err
		}
		diagram = Diagram{IsSequence: true, Sequence: parsed}
	} else {
		return nil, errors.New("unknown diagram type")
	}

	return &diagram, nil
}
