package cli

import "strings"

// reorderFlagsFirst moves CLI flags before positional arguments so parsing works
// regardless of whether users pass `./path --output ./out` or `--output ./out ./path`.
func reorderFlagsFirst(args []string, boolFlags map[string]bool) []string {
	if len(args) == 0 {
		return args
	}

	flagArgs := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !isFlagToken(arg) {
			positionals = append(positionals, arg)
			continue
		}

		flagArgs = append(flagArgs, arg)
		name, _, inlineValue := parseFlagToken(arg)
		if inlineValue || boolFlags[name] {
			continue
		}

		if i+1 < len(args) && !isFlagToken(args[i+1]) {
			flagArgs = append(flagArgs, args[i+1])
			i++
		}
	}

	return append(flagArgs, positionals...)
}

func isFlagToken(arg string) bool {
	if !strings.HasPrefix(arg, "-") {
		return false
	}
	if strings.HasPrefix(arg, "--") {
		return len(arg) > 2
	}
	return len(arg) > 1
}

func parseFlagToken(arg string) (name string, value string, hasInlineValue bool) {
	trimmed := arg
	switch {
	case strings.HasPrefix(arg, "--"):
		trimmed = strings.TrimPrefix(arg, "--")
	case strings.HasPrefix(arg, "-"):
		trimmed = strings.TrimPrefix(arg, "-")
	}

	if idx := strings.Index(trimmed, "="); idx >= 0 {
		return trimmed[:idx], trimmed[idx+1:], true
	}
	return trimmed, "", false
}

var scanBoolFlags = map[string]bool{
	"include-vulnerabilities": true,
	"fail-on-vuln":            true,
}

var githubBoolFlags = map[string]bool{
	"health":                  true,
	"include-vulnerabilities": true,
	"fail-on-vuln":            true,
}
