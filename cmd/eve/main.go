package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nhestrompia/eve"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "canonicalize":
		return runCanonicalize(args[1:], stdout, stderr)
	case "version":
		if len(args) != 1 {
			fmt.Fprintln(stderr, "eve version takes no arguments")
			return 2
		}
		fmt.Fprintf(stdout, "eve %s (protocol v%d)\n", eve.CLIVersion, eve.ProtocolVersion)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runValidate(paths []string, stdout io.Writer, stderr io.Writer) int {
	if len(paths) == 0 {
		fmt.Fprintln(stderr, "eve validate requires at least one file")
		return 2
	}

	exitCode := 0
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", path, err)
			exitCode = 2
			continue
		}

		if _, err := eve.Parse(data); err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", path, err)
			if exitCode != 2 {
				exitCode = 1
			}
			continue
		}
		fmt.Fprintf(stdout, "%s: valid\n", path)
	}

	return exitCode
}

func runCanonicalize(paths []string, stdout io.Writer, stderr io.Writer) int {
	if len(paths) != 1 {
		fmt.Fprintln(stderr, "eve canonicalize requires exactly one file")
		return 2
	}

	data, err := os.ReadFile(paths[0])
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", paths[0], err)
		return 2
	}

	evolution, err := eve.Parse(data)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", paths[0], err)
		return 1
	}

	canonical, err := eve.CanonicalJSON(evolution)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", paths[0], err)
		return 1
	}

	fmt.Fprintln(stdout, string(canonical))
	return 0
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  eve validate <file...>")
	fmt.Fprintln(w, "  eve canonicalize <file>")
	fmt.Fprintln(w, "  eve version")
}
