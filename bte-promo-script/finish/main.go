// Command promo-finish is the orchestrator tool installed on polecat-agent.
// Claude runs these subcommands — never GenAxis/S3 HTTP directly.
//
//	promo-finish upload key=path [key=path ...]  → stdout: { artifacts: [{key, file_url, …}] }
//	promo-finish webhook '<upload-json>'          → one GenAxis completed webhook
//	promo-finish finish key=path [key=path ...]   → upload all + webhook once
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "upload":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		err = runUpload(os.Args[2:]...)
	case "webhook":
		if len(os.Args) != 3 {
			usage()
			os.Exit(2)
		}
		err = runWebhook(os.Args[2])
	case "finish":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		err = runFinish(os.Args[2:]...)
	case "-h", "--help", "help":
		usage()
		return
	default:
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "promo-finish: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "PROMO_FINISH_OK")
}

func usage() {
	fmt.Fprintf(os.Stderr, `promo-finish — orchestrator handoff tool (runs on Polecat)

Env (set by operator): PROMO_API_URL, PROMO_JOB_NAME, GENAXIS_REQUEST_ID

Commands:
  upload key=path [key=path ...]   Upload labeled outputs → stdout JSON { artifacts: [...] }
  webhook '<upload-json>'          One backend webhook with all script URLs
  finish key=path [key=path ...]   upload + webhook in one step

Keys must match repos/bte-promo-script-repo/scripts/PROMO_FINISH.md (which step produced which file).

Example:
  RESULT=$(promo-finish upload \
    script=output/script.md \
    metadata=output/metadata.json \
    beats=output/beats.json)
  promo-finish webhook "$RESULT"

One step:
  promo-finish finish script=output/script.md metadata=output/metadata.json
`)
}
