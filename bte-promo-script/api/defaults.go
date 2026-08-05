package main

import "strings"

// Non-secret promo-api constants. Change here, then rebuild promo-api.
const (
	DefaultGitRepo     = "git@github.com:priyanshur01/gastown-static.git"
	DefaultGitBranch   = "main"
	DefaultRigName    = "promo-script-tool"
	DefaultGenAxisURL = "https://gen-axis.pocketfm.com"
	GenAxisWebhookPath  = "/v1/bte/internal/webhook"
	// In-cluster promo-api URL used by Polecats for design-B handoff.
	DefaultPromoAPIURL = "http://promo-api.gastown-system.svc:8080"

	// GenAxis webhook event types (body.type).
	GenAxisTypePromoStarted   = "PROMO_SCRIPT_STARTED"
	GenAxisTypePromoCompleted = "PROMO_SCRIPT_COMPLETED"

	annotationRequestID = "gastown.io/request-id"

	// S3 not wired yet — upload handler logs paths and returns these stub URLs.
	StubFileURLBase = "https://promo-stub.pocketfm.test/files"
)

func genAxisWebhookURL() string {
	return strings.TrimRight(DefaultGenAxisURL, "/") + GenAxisWebhookPath
}
