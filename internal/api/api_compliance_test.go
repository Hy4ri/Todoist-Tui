package api_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPICompliance(t *testing.T) {
	// allowedURLs is a list of approved URL prefixes.
	// strictly standardizing on API v1 https://api.todoist.com/api/v1 // and OAuth.
	allowedURLs := []string{
		"https://api.todoist.com/api/v1",
		"https://todoist.com/oauth/authorize",
		"https://todoist.com/oauth/access_token",
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// We want to scan the "internal" directory.
	// Strategy: find the project root by looking for "internal" in the path or just assume relative paths if running from root.

	var internalDir string
	if strings.HasSuffix(cwd, "api") && strings.Contains(cwd, "internal") {
		// Running from internal/api
		internalDir = filepath.Join(cwd, "..")
	} else if strings.Contains(cwd, "todoist") {
		// Try to find internal dir from current location
		candidates := []string{
			"internal",
			"./internal",
			"../internal",
			"../../internal",
		}
		for _, c := range candidates {
			abs, _ := filepath.Abs(c)
			if stat, err := os.Stat(abs); err == nil && stat.IsDir() {
				internalDir = abs
				break
			}
		}
	}

	if internalDir == "" {
		t.Logf("Could not locate internal directory from %s, skipping scan", cwd)
		return
	}

	err = filepath.Walk(internalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only check Go files
		if !strings.HasSuffix(info.Name(), ".go") {
			return nil
		}

		// Skip verify_test or compliance test to avoid self-flagging
		if strings.Contains(path, "api_comp_test.go") || strings.Contains(path, "api_compliance_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		lineNumber := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lineNumber++
			line := scanner.Text()

			// Simple check for https://
			if idx := strings.Index(line, "https://"); idx != -1 {
				// Clean the line to just the part starting with https://
				rest := line[idx:]

				allowed := false
				for _, p := range allowedURLs {
					if strings.HasPrefix(rest, p) {
						allowed = true
						break
					}
				}

				if !allowed {
					// Extract the URL for the error message (rough extraction)
					splits := strings.FieldsFunc(rest, func(r rune) bool {
						return r == '"' || r == '`' || r == ' ' || r == '>' || r == ')'
					})
					url := ""
					if len(splits) > 0 {
						url = splits[0]
					} else {
						url = rest
					}

					t.Errorf("Unauthorized URL found in %s:%d: %s", path, lineNumber, url)
				}
			}
		}

		return scanner.Err()
	})

	if err != nil {
		t.Fatalf("Failed to walk directories: %v", err)
	}
}
