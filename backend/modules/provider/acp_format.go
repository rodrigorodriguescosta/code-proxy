package provider

import (
	"fmt"
	"strings"

	acp "github.com/coder/acp-go-sdk"
)

// formatACPToolCall converts an ACP ToolCall into formatted Markdown
// Aligned with claude.go formatToolUse for visual consistency in Cursor
func formatACPToolCall(tc *acp.SessionUpdateToolCall) string {
	var parts []string

	// Rich content: diffs, terminals, text
	for _, c := range tc.Content {
		switch {
		case c.Diff != nil:
			path := shortPath(c.Diff.Path)
			old := ptrStr(c.Diff.OldText)
			added, removed := countDiffLines(old, c.Diff.NewText)
			stats := diffStats(added, removed)

			oldDiff := prefixDiffLines(old, "- ")
			newDiff := prefixDiffLines(c.Diff.NewText, "+ ")

			lang := detectLang(c.Diff.Path)
			var b strings.Builder
			b.WriteString(fmt.Sprintf("#### Edit `%s` %s\n", path, stats))
			b.WriteString(fmt.Sprintf("```diff\n%s%s```\n", oldDiff, newDiff))
			b.WriteString(fmt.Sprintf("```%s:%s\n%s\n```", lang, c.Diff.Path, c.Diff.NewText))
			parts = append(parts, b.String())

		case c.Terminal != nil:
			// Terminal inline: the title usually contains the command
			if tc.Title != "" {
				parts = append(parts, fmt.Sprintf("#### %s\n```bash\n%s\n```", tc.Title, tc.Title))
			} else {
				parts = append(parts, fmt.Sprintf("*Terminal: `%s`*", c.Terminal.TerminalId))
			}

		case c.Content != nil:
			if c.Content.Content.Text != nil {
				parts = append(parts, c.Content.Content.Text.Text)
			}
		}
	}

	// File locations (for reads/edits without content)
	if len(parts) == 0 {
		for _, loc := range tc.Locations {
			path := shortPath(loc.Path)
			switch tc.Kind {
			case acp.ToolKindRead:
				parts = append(parts, fmt.Sprintf("*Read `%s`*", path))
			case acp.ToolKindEdit:
				parts = append(parts, fmt.Sprintf("*Edit `%s`*", path))
			default:
				parts = append(parts, fmt.Sprintf("*File `%s`*", path))
			}
		}
	}

	if len(parts) > 0 {
		return strings.Join(parts, "\n\n")
	}

	// Fallback: title
	if tc.Title != "" {
		switch tc.Kind {
		case acp.ToolKindExecute:
			return fmt.Sprintf("#### %s\n```bash\n%s\n```", tc.Title, tc.Title)
		case acp.ToolKindEdit:
			return fmt.Sprintf("*Edit: %s*", tc.Title)
		case acp.ToolKindRead:
			return fmt.Sprintf("*Read: %s*", tc.Title)
		default:
			return fmt.Sprintf("*%s*", tc.Title)
		}
	}

	return ""
}

// formatACPToolCallUpdate converts tool call updates into text (only failures)
func formatACPToolCallUpdate(tc *acp.SessionToolCallUpdate) string {
	if tc.Status != nil && *tc.Status == acp.ToolCallStatusFailed {
		title := string(tc.ToolCallId)
		if tc.Title != nil {
			title = *tc.Title
		}
		return fmt.Sprintf("**Tool failed:** %s", title)
	}
	return ""
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
