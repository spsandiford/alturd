package git

// ParseRefArgs converts cobra's positional args (after flag parsing) to the
// git diff argument slice that ExecRunner.Run expects.
//
// dashIdx is cobra's cmd.ArgsLenAtDash() — the index in args where the "--"
// path separator appeared in the original os.Args, or -1 when "--" was absent.
// Cobra strips the "--" separator itself and reports only the split index; this
// function re-inserts the separator between ref args and path args so that git
// receives the correct argument grammar.
//
// ParseRefArgs is a verbatim pass-through of ref tokens: it does not parse or
// validate ref strings (git itself handles two-dot / three-dot / two-arg
// semantic differences). The caller prepends the "diff" subcommand before
// passing the result to Runner.Run.
//
// Examples:
//
//	ParseRefArgs([], -1)                  → []
//	ParseRefArgs(["HEAD~1"], -1)          → ["HEAD~1"]
//	ParseRefArgs(["HEAD~1..HEAD"], -1)    → ["HEAD~1..HEAD"]
//	ParseRefArgs(["main...feature"], -1)  → ["main...feature"]
//	ParseRefArgs(["HEAD~1","HEAD"], -1)   → ["HEAD~1","HEAD"]
//	ParseRefArgs(["internal/"], 0)        → ["--","internal/"]
//	ParseRefArgs(["HEAD~1","internal/"],1)→ ["HEAD~1","--","internal/"]
func ParseRefArgs(args []string, dashIdx int) []string {
	var refs []string
	var paths []string

	if dashIdx >= 0 {
		refs = args[:dashIdx]
		paths = args[dashIdx:]
	} else {
		refs = args
	}

	// Pre-allocate: refs + optional "--" separator + paths.
	extra := 0
	if len(paths) > 0 {
		extra = 1
	}
	result := make([]string, 0, len(refs)+extra+len(paths))
	result = append(result, refs...)
	if len(paths) > 0 {
		// Re-insert the "--" separator that cobra stripped (Pitfall 2).
		// Without this, git diff would treat path args as refs.
		result = append(result, "--")
		result = append(result, paths...)
	}
	return result
}
