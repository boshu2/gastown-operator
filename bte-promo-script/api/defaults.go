package main

import "strings"

// Non-secret promo-api constants. Override via env in production manifests.
const (
	DefaultNamespace = "bte-gastown"

	DefaultRigName = "promo-script-tool"
	// Rig metadata only — Polecats use skipGitInit and do not clone a git repo.
	DefaultRigGitURL = "file:///promo-tool"
	// Static promo pipeline workspace inside the polecat-agent image (writable copy on emptyDir).
	DefaultPromoWorkspacePath = "/workspace/promo-tool"

	DefaultGenAxisURL  = "https://gen-axis.pocketfm.com"
	GenAxisWebhookPath = "/v1/bte/internal/webhook"

	// In-cluster service DNS (Pod 2 → Pod 1).
	DefaultPromoAPIURL      = "http://promo-api.bte-gastown.svc:8080"
	DefaultMadEyeProxyURL   = "http://madeye-proxy.bte-gastown.svc:4000"
	DefaultMadEyeProxyAuthSecret = "madeye-proxy-auth"
	DefaultMadEyeProxyAuthKey    = "master-key"

	// GenAxis webhook event types (body.type).
	GenAxisTypePromoStarted   = "PROMO_SCRIPT_STARTED"
	GenAxisTypePromoCompleted = "PROMO_SCRIPT_COMPLETED"

	annotationRequestID = "gastown.io/request-id"

	// Dev-only: hostPath for local W4 export. Empty disables mount in polecat pods.
	DefaultDevOutputHostPath  = ""
	DefaultDevOutputMountPath = "/workspace/dev-output"
	DefaultDevSaveURL         = "" // local: http://host.docker.internal:3099/dev-save
	AnnotationDevOutputHost   = "gastown.io/dev-output-host"

	// S3 not wired yet — upload handler logs paths and returns stub URLs.
	StubFileURLBase = "https://promo-stub.pocketfm.test/files"

	// Local dev only (deploy-infra.sh / apply-secrets.sh). Prod uses ExternalSecret + madeye-proxy-auth.
	DefaultMadEyeUserEmail = "promo@pocketfm.com"
	DefaultProxyMasterKey  = "sk-local-madeye-proxy"
)

func genAxisWebhookURL() string {
	return strings.TrimRight(DefaultGenAxisURL, "/") + GenAxisWebhookPath
}
