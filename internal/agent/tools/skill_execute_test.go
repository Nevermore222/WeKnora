package tools

import (
	"strings"
	"testing"
)

func TestFailedScriptErrorSummaryIncludesStderrAndStdout(t *testing.T) {
	summary := failedScriptErrorSummary(
		1,
		"",
		"created formulas before commit",
		"target file is locked and could not be replaced. Close the workbook in Office/WPS and retry.",
	)

	if !strings.Contains(summary, "Script exited with code 1") {
		t.Fatalf("summary missing exit code: %q", summary)
	}
	if !strings.Contains(summary, "stderr: target file is locked") {
		t.Fatalf("summary missing stderr: %q", summary)
	}
	if !strings.Contains(summary, "stdout: created formulas before commit") {
		t.Fatalf("summary missing stdout: %q", summary)
	}
}

func TestFailedScriptErrorSummaryTruncatesLongStderr(t *testing.T) {
	longStderr := strings.Repeat("x", 1400)
	summary := failedScriptErrorSummary(1, "", "", longStderr)

	if len(summary) > 1300 {
		t.Fatalf("summary was not compacted: len=%d", len(summary))
	}
	if !strings.HasSuffix(summary, "...") {
		t.Fatalf("summary should end with ellipsis: %q", summary)
	}
}
