// Command envtojson reads a .env file and emits the flat JSON object consumed by
// the PR acceptance workflow (.github/workflows/pr-acceptance.yml) — i.e. the
// env.json blob you paste into the 1Password item.
//
// It parses .env with the same library the acceptance tests use (joho/godotenv),
// so the JSON contains exactly the values the tests would see. By default it keeps
// only the credentials CI needs (STREAMKAP_* and TF_VAR_*, non-empty) and drops
// local-only control vars (TF_ACC, STREAMKAP_BACKEND_PATH, UPDATE_*, TF_LOG).
//
// Usage:
//
//	go run ./cmd/envtojson                 # read ./.env, print JSON to stdout
//	go run ./cmd/envtojson -in path/.env   # read a specific file
//	go run ./cmd/envtojson -o env.json     # write to a file
//	go run ./cmd/envtojson -all            # include every key (no filtering)
//	go run ./cmd/envtojson -include-empty   # keep keys with empty values too
//	go run ./cmd/envtojson -from-env       # overlay live env (fills STREAMKAP_CLIENT_ID etc.)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// excluded lists keys that match the credential prefixes but are local-only
// control knobs, not credentials, so they never belong in env.json.
var excluded = map[string]bool{
	"STREAMKAP_BACKEND_PATH": true,
}

func keep(key, value string, all, includeEmpty bool) bool {
	if !includeEmpty && value == "" {
		return false
	}
	if all {
		return true
	}
	if excluded[key] {
		return false
	}
	return strings.HasPrefix(key, "TF_VAR_") || strings.HasPrefix(key, "STREAMKAP_")
}

func main() {
	in := flag.String("in", ".env", "path to the .env file to read")
	out := flag.String("o", "", "output file (default: stdout)")
	all := flag.Bool("all", false, "include every key, not just STREAMKAP_*/TF_VAR_*")
	includeEmpty := flag.Bool("include-empty", false, "keep keys whose value is empty")
	compact := flag.Bool("compact", false, "emit compact single-line JSON instead of indented")
	fromEnv := flag.Bool("from-env", false, "overlay live environment variables on top of .env (env wins) — fills creds the .env lacks, e.g. STREAMKAP_CLIENT_ID")
	flag.Parse()

	raw, err := godotenv.Read(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envtojson: reading %s: %v\n", *in, err)
		os.Exit(1)
	}

	selected := make(map[string]string, len(raw))
	for k, v := range raw {
		if keep(k, v, *all, *includeEmpty) {
			selected[k] = v
		}
	}

	// With -from-env, the live process environment is overlaid on top of .env,
	// so vars exported in the shell (commonly STREAMKAP_CLIENT_ID/SECRET/HOST,
	// which the acceptance precheck requires) are included even when absent
	// from .env. Live values win on conflict.
	if *fromEnv {
		for _, kv := range os.Environ() {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				continue
			}
			if keep(k, v, *all, *includeEmpty) {
				selected[k] = v
			}
		}
	}

	if len(selected) == 0 {
		fmt.Fprintf(os.Stderr, "envtojson: no matching keys found in %s\n", *in)
		os.Exit(1)
	}

	// encoding/json sorts map keys, so output is stable/diffable.
	enc := func() ([]byte, error) {
		if *compact {
			return json.Marshal(selected)
		}
		return json.MarshalIndent(selected, "", "  ")
	}
	data, err := enc()
	if err != nil {
		fmt.Fprintf(os.Stderr, "envtojson: encoding json: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if *out == "" {
		os.Stdout.Write(data)
	} else if err := os.WriteFile(*out, data, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "envtojson: writing %s: %v\n", *out, err)
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "envtojson: wrote %d keys to %s\n", len(selected), *out)
	}
}
