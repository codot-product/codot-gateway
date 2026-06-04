package context

import (
	"bufio"
	"os"
	"strings"
)

// Symbol represents an AST symbol found in code.
type Symbol struct {
	Name string
	Type string
	Body string
}

// ASTParser wrapper to satisfy cache.go
type ASTParser struct{}

// NewASTParser creates a new ASTParser
func NewASTParser() *ASTParser {
	return &ASTParser{}
}

// Parse stub to satisfy cache.go
func (p *ASTParser) Parse(filename string, code []byte) ([]Symbol, error) {
	return nil, nil
}

// GenerateArchitectureBrief reads a file and returns its structural map outline
func GenerateArchitectureBrief(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var brief strings.Builder
	scanner := bufio.NewScanner(file)

	brief.WriteString("File: " + filePath + "\n")
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Capture function boundaries across Go, JS, TypeScript, and Python signatures
		if strings.HasPrefix(line, "func ") || 
		   strings.HasPrefix(line, "function ") || 
		   (strings.HasPrefix(line, "export const ") && strings.Contains(line, "=>")) ||
		   strings.HasPrefix(line, "def ") {
			
			// Store only the structural signature line, ignoring the inner code block logic!
			brief.WriteString("  " + line + "\n")
		}
	}

	return brief.String(), nil
}
