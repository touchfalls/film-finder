package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadDotenv reads KEY=value lines from .env in the current directory and next
// to the binary. Variables already set in the environment win.
func loadDotenv() {
	paths := []string{".env"}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ".env"))
	}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			k, v, ok := parseEnvLine(sc.Text())
			if _, set := os.LookupEnv(k); ok && !set {
				os.Setenv(k, v)
			}
		}
		f.Close()
	}
}

// shortcut: no multiline values or ${VAR} expansion, switch to godotenv if needed
func parseEnvLine(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "export "))
	if line == "" || line[0] == '#' {
		return "", "", false
	}
	key, val, ok = strings.Cut(line, "=")
	key, val = strings.TrimSpace(key), strings.TrimSpace(val)
	if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
		val = val[1 : len(val)-1]
	}
	return key, val, ok && key != ""
}
