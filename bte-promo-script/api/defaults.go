package main

import "strings"

// Non-secret promo-api constants. Change here, then rebuild promo-api.
const (
	DefaultRigName = "promo-script-tool"
	// Rig metadata only — Polecats use skipGitInit and do not clone a git repo.
	DefaultRigGitURL = "file:///promo-tool"
	// Static promo pipeline workspace inside the polecat-agent image (writable copy on emptyDir).
	DefaultPromoWorkspacePath = "/workspace/promo-tool"
	DefaultGenAxisURL = "https://gen-axis.pocketfm.com"
	GenAxisWebhookPath  = "/v1/bte/internal/webhook"
	// In-cluster promo-api URL used by Polecats for design-B handoff.
	DefaultPromoAPIURL = "http://promo-api.gastown-system.svc:8080"
	// In-cluster Anthropic-compatible proxy → MadEye (our code, bte-promo-script/madeye-proxy).
	DefaultMadEyeProxyURL = "http://madeye-proxy.gastown-system.svc:4000"
	// MadEye proxy constants (not K8s-injected — change here, redeploy madeye-proxy stack).
	DefaultMadEyeUserEmail = "priyanshu.rajput@pocketfm.com"
	DefaultProxyMasterKey  = "sk-local-madeye-proxy"

	// GenAxis webhook event types (body.type).
	GenAxisTypePromoStarted   = "PROMO_SCRIPT_STARTED"
	GenAxisTypePromoCompleted = "PROMO_SCRIPT_COMPLETED"

	annotationRequestID = "gastown.io/request-id"

	// LOCAL TEST ONLY — remove before push. Host path on Docker Desktop Mac where W4
	// artifacts are copied by promo-finish. Empty string disables the mount.
	DefaultDevOutputHostPath = "/Users/priyansh.rajput/Desktop/bte-script"
	// Host HTTP endpoint for W4 script export (works on Docker Desktop Mac; hostPath does not).
	DefaultDevSaveURL = "http://host.docker.internal:9999/v1/bte/internal/dev-save"
	// In-pod mount; must match pkg/pod builder DevOutputMountPath.
	DefaultDevOutputMountPath = "/workspace/dev-output"
	AnnotationDevOutputHost   = "gastown.io/dev-output-host"

	// S3 not wired yet — upload handler logs paths and returns these stub URLs.
	StubFileURLBase = "https://promo-stub.pocketfm.test/files"
)

func genAxisWebhookURL() string {
	return strings.TrimRight(DefaultGenAxisURL, "/") + GenAxisWebhookPath
}
