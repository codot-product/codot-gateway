package metrics

import (
	"regexp"
	"strings"
	"time"

	"github.com/codot-product/codot-gateway/internal/db"
)

// Auditor audits code snippets for quality, performance, and security issues.
type Auditor struct {
	secretRegexes   map[string]*regexp.Regexp
	sqlInjection    *regexp.Regexp
	cmdInjection    *regexp.Regexp
	todoChecker     *regexp.Regexp
	emptyCatchRegex *regexp.Regexp
}

// NewAuditor creates a new Auditor instance with compiled rules.
func NewAuditor() *Auditor {
	return &Auditor{
		secretRegexes: map[string]*regexp.Regexp{
			"Generic API Key":  regexp.MustCompile(`(?i)(api[-_]?key|secret[-_]?key|auth[-_]?token|password|passwd|private[-_]?key)\s*[:=]\s*['"][a-zA-Z0-9_\-\.\~]{16,}['"]`),
			"AWS Access Key":   regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			"AWS Secret Key":   regexp.MustCompile(`(?i)aws[-_]?secret.*\s*[:=]\s*['"][a-zA-Z0-9/\+]{40}[']`),
			"Bearer Token":     regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.\~]{20,}`),
			"Slack Webhook":    regexp.MustCompile(`https://hooks\.slack\.com/services/T[a-zA-Z0-9_]+/B[a-zA-Z0-9_]+/[a-zA-Z0-9_]+`),
			"Private SSH Key":  regexp.MustCompile(`-----BEGIN [A-Z]+ PRIVATE KEY-----`),
		},
		sqlInjection:    regexp.MustCompile(`(?i)(select|insert|update|delete)\s+.*\s+from\s+.*\s+where\s+.*=\s*(\+.*\+.*\+|\+.*|\$\{[a-zA-Z0-9_]+\}|%s|fmt\.Sprintf|f['"].*\{)`),
		cmdInjection:    regexp.MustCompile(`(?i)(exec\.(Command|CommandContext)|os\.system|subprocess\.(Popen|run|call)|child_process\.(exec|spawn)|eval\()`),
		todoChecker:     regexp.MustCompile(`(?i)\/\/\s*TODO|\#\s*TODO|\/\*\s*TODO`),
		emptyCatchRegex: regexp.MustCompile(`(?s)(catch\s*\(.*?\)\s*\{\s*\}|except\s*:\s*pass|except\s+[a-zA-Z0-9_]+\s*:\s*pass)`),
	}
}

// AuditContent parses markdown code blocks and runs checks on them.
func (a *Auditor) AuditContent(requestID string, content string) []db.AuditLog {
	var violations []db.AuditLog

	// Find markdown code blocks: ```lang\n [code] \n```
	codeBlockRegex := regexp.MustCompile("(?s)```([a-zA-Z0-9_-]*)\\n(.*?)\\n```")
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		lang := strings.ToLower(match[1])
		code := match[2]

		// Run rules
		violations = append(violations, a.checkSecrets(requestID, lang, code)...)
		violations = append(violations, a.checkSecurity(requestID, lang, code)...)
		violations = append(violations, a.checkQuality(requestID, lang, code)...)
	}

	return violations
}

// checkSecrets searches for hardcoded API keys and secrets.
func (a *Auditor) checkSecrets(requestID, lang, code string) []db.AuditLog {
	var logs []db.AuditLog

	for ruleName, re := range a.secretRegexes {
		if loc := re.FindStringIndex(code); loc != nil {
			snippet := extractSnippet(code, loc[0], loc[1])
			// Mask actual secret in snippet
			maskedSnippet := maskSecret(snippet)

			logs = append(logs, db.AuditLog{
				RequestID: requestID,
				Timestamp: time.Now(),
				FileType:  lang,
				Severity:  "critical",
				RuleName:  ruleName,
				Message:   "Hardcoded credentials/secrets found in generated code.",
				Snippet:   maskedSnippet,
			})
		}
	}

	return logs
}

// checkSecurity searches for SQL/Command Injections.
func (a *Auditor) checkSecurity(requestID, lang, code string) []db.AuditLog {
	var logs []db.AuditLog

	// Check SQL Injection
	if loc := a.sqlInjection.FindStringIndex(code); loc != nil {
		logs = append(logs, db.AuditLog{
			RequestID: requestID,
			Timestamp: time.Now(),
			FileType:  lang,
			Severity:  "high",
			RuleName:  "SQL Injection Risk",
			Message:   "Potential SQL Injection vulnerability detected. Use parameterized queries instead of string formatting.",
			Snippet:   extractSnippet(code, loc[0], loc[1]),
		})
	}

	// Check Command Injection / Eval
	if loc := a.cmdInjection.FindStringIndex(code); loc != nil {
		logs = append(logs, db.AuditLog{
			RequestID: requestID,
			Timestamp: time.Now(),
			FileType:  lang,
			Severity:  "high",
			RuleName:  "Command Injection / Eval Risk",
			Message:   "Vulnerable OS command execution or eval usage detected. Ensure input validation is strictly performed.",
			Snippet:   extractSnippet(code, loc[0], loc[1]),
		})
	}

	return logs
}

// checkQuality searches for code quality and syntax issues.
func (a *Auditor) checkQuality(requestID, lang, code string) []db.AuditLog {
	var logs []db.AuditLog

	// Check for empty catch/except blocks
	if loc := a.emptyCatchRegex.FindStringIndex(code); loc != nil {
		logs = append(logs, db.AuditLog{
			RequestID: requestID,
			Timestamp: time.Now(),
			FileType:  lang,
			Severity:  "medium",
			RuleName:  "Empty Catch / Ignored Error",
			Message:   "Error/Exception is caught but ignored. Log or handle errors properly.",
			Snippet:   extractSnippet(code, loc[0], loc[1]),
		})
	}

	// Check for TODOs
	if loc := a.todoChecker.FindStringIndex(code); loc != nil {
		logs = append(logs, db.AuditLog{
			RequestID: requestID,
			Timestamp: time.Now(),
			FileType:  lang,
			Severity:  "low",
			RuleName:  "TODO Leftover",
			Message:   "Unfinished code: TODO comment found in the generated snippet.",
			Snippet:   extractSnippet(code, loc[0], loc[1]),
		})
	}

	return logs
}

// Helper to extract a snippet around the violation
func extractSnippet(code string, start, end int) string {
	// Expand start and end to line boundaries
	startIdx := start - 20
	if startIdx < 0 {
		startIdx = 0
	}
	for startIdx > 0 && code[startIdx] != '\n' {
		startIdx--
	}
	if code[startIdx] == '\n' {
		startIdx++
	}

	endIdx := end + 40
	if endIdx > len(code) {
		endIdx = len(code)
	}
	for endIdx < len(code) && code[endIdx] != '\n' {
		endIdx++
	}

	snippet := code[startIdx:endIdx]
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return strings.TrimSpace(snippet)
}

// Helper to mask secret strings in logs to prevent logging credentials
func maskSecret(snippet string) string {
	parts := strings.SplitN(snippet, "=", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(snippet, ":", 2)
	}
	if len(parts) == 2 {
		val := strings.TrimSpace(parts[1])
		if len(val) > 4 {
			quote := ""
			if strings.HasPrefix(val, "'") || strings.HasPrefix(val, "\"") {
				quote = string(val[0])
			}
			masked := quote + "********" + quote
			return strings.TrimSpace(parts[0]) + " = " + masked
		}
	}
	return "[Sensitive Credential Masked]"
}
