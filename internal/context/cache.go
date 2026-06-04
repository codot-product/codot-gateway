package context

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/codot-product/codot-gateway/api/openai"
)

// ContextPruner handles prompt minimization and token saving.
type ContextPruner struct {
	parser *ASTParser
}

// NewContextPruner creates a new ContextPruner instance.
func NewContextPruner() *ContextPruner {
	return &ContextPruner{
		parser: NewASTParser(),
	}
}

// PruneRequest takes a ChatCompletionRequest, compresses embedded code snippets and prunes message history,
// and returns the modified request and the approximate number of characters/tokens saved.
func (cp *ContextPruner) PruneRequest(req *openai.ChatCompletionRequest, maxMessageHistory int, maxCodeBlockLines int) (openai.ChatCompletionRequest, int) {
	if req == nil {
		return openai.ChatCompletionRequest{}, 0
	}

	// Deep copy request
	newReq := *req
	newReq.Messages = make([]openai.Message, len(req.Messages))
	copy(newReq.Messages, req.Messages)

	totalSavedChars := 0

	// 1. Compress code blocks in the messages
	for i, msg := range newReq.Messages {
		compressedContent, saved := cp.pruneCodeBlocks(msg.Content, maxCodeBlockLines)
		newReq.Messages[i].Content = compressedContent
		totalSavedChars += saved
	}

	// 2. Prune old history if it exceeds maxMessageHistory
	// Always keep the system message (typically index 0 or role == "system")
	// and keep the most recent messages up to maxMessageHistory.
	if len(newReq.Messages) > maxMessageHistory {
		var prunedMessages []openai.Message
		var systemMessage *openai.Message

		// Find system message
		for _, msg := range newReq.Messages {
			if msg.Role == "system" {
				systemMessage = &msg
				break
			}
		}

		// Calculate start index for recent messages
		startIndex := len(newReq.Messages) - maxMessageHistory
		if startIndex < 0 {
			startIndex = 0
		}

		if systemMessage != nil && startIndex > 0 {
			prunedMessages = append(prunedMessages, *systemMessage)
			// Deduplicate if system message is also in the tail
			for j := startIndex; j < len(newReq.Messages); j++ {
				if newReq.Messages[j].Role == "system" {
					continue
				}
				prunedMessages = append(prunedMessages, newReq.Messages[j])
			}
		} else {
			prunedMessages = newReq.Messages[startIndex:]
		}

		// Count characters of removed messages
		removedChars := 0
		oldMsgCount := len(newReq.Messages)
		newMsgCount := len(prunedMessages)
		if oldMsgCount > newMsgCount {
			// Find which messages were discarded and sum their lengths
			// Simple approximation: sum of discarded lengths
			// In our case, we reconstruct the lists. Let's calculate:
			oldLen := 0
			for _, m := range newReq.Messages {
				oldLen += len(m.Content)
			}
			newLen := 0
			for _, m := range prunedMessages {
				newLen += len(m.Content)
			}
			removedChars = oldLen - newLen
		}

		newReq.Messages = prunedMessages
		totalSavedChars += removedChars
	}

	// Approximation: 1 token ~= 4 characters for English text
	tokensSaved := totalSavedChars / 4
	return newReq, tokensSaved
}

// pruneCodeBlocks looks for markdown code blocks (e.g. ```go ... ```) and prunes them if they exceed lines threshold.
func (cp *ContextPruner) pruneCodeBlocks(content string, maxLines int) (string, int) {
	// Match code blocks: ```lang\n [code] \n```
	codeBlockRegex := regexp.MustCompile("(?s)```([a-zA-Z0-9_-]*)\\n(.*?)\\n```")
	savedChars := 0

	result := codeBlockRegex.ReplaceAllStringFunc(content, func(match string) string {
		submatches := codeBlockRegex.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}
		lang := submatches[1]
		code := submatches[2]

		lineCount := strings.Count(code, "\n") + 1
		if lineCount <= maxLines {
			return match
		}

		// Too long! Minimize this code block.
		filename := "snippet." + lang
		if lang == "" {
			filename = "snippet.txt"
		}

		symbols, err := cp.parser.Parse(filename, []byte(code))
		if err != nil || len(symbols) == 0 {
			// Fallback: just keep the first and last few lines
			lines := strings.Split(code, "\n")
			prunedCode := strings.Join(lines[:maxLines/2], "\n") +
				fmt.Sprintf("\n\n// ... [Pruned %d lines of code for context optimization] ...\n\n", lineCount-maxLines) +
				strings.Join(lines[len(lines)-maxLines/2:], "\n")
			
			diff := len(match) - len(fmt.Sprintf("```%s\n%s\n```", lang, prunedCode))
			if diff > 0 {
				savedChars += diff
			}
			return fmt.Sprintf("```%s\n%s\n```", lang, prunedCode)
		}

		// Rebuild code structure from AST symbols (keeps declarations and removes implementation details)
		var skeleton strings.Builder
		skeleton.WriteString(fmt.Sprintf("// [Optimized Context: AST Structure Only - %d lines pruned]\n", lineCount))
		
		var imports []string
		var structures []string
		var functions []string

		for _, sym := range symbols {
			switch sym.Type {
			case "import":
				imports = append(imports, sym.Body)
			case "struct", "class", "interface":
				// For classes/structs, we can list the signature or truncated block
				lines := strings.Split(sym.Body, "\n")
				sig := lines[0]
				if len(lines) > 1 {
					sig += "\n  // ... (properties/methods pruned) ..."
				}
				structures = append(structures, sig)
			case "function", "method":
				// For functions, we extract the signature line (first line of function declaration)
				lines := strings.Split(sym.Body, "\n")
				sig := lines[0]
				if !strings.HasSuffix(strings.TrimSpace(sig), ";") && !strings.HasSuffix(strings.TrimSpace(sig), "{") {
					// Add placeholder body representation depending on language
					if lang == "py" {
						sig += "\n    pass"
					} else {
						sig += " { /* implementation pruned */ }"
					}
				} else if strings.HasSuffix(strings.TrimSpace(sig), "{") {
					sig = strings.TrimSuffix(strings.TrimSpace(sig), "{")
					if lang == "py" {
						sig += "\n    pass"
					} else {
						sig += " { /* implementation pruned */ }"
					}
				}
				functions = append(functions, sig)
			}
		}

		if len(imports) > 0 {
			skeleton.WriteString("\n// Imports:\n")
			skeleton.WriteString(strings.Join(imports, "\n") + "\n")
		}
		if len(structures) > 0 {
			skeleton.WriteString("\n// Structures/Classes:\n")
			skeleton.WriteString(strings.Join(structures, "\n") + "\n")
		}
		if len(functions) > 0 {
			skeleton.WriteString("\n// Functions/API Signatures:\n")
			skeleton.WriteString(strings.Join(functions, "\n") + "\n")
		}

		prunedResult := fmt.Sprintf("```%s\n%s```", lang, skeleton.String())
		diff := len(match) - len(prunedResult)
		if diff > 0 {
			savedChars += diff
			return prunedResult
		}
		return match
	})

	return result, savedChars
}
