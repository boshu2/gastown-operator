package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

func main() {
	cfg := loadConfig()
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gastownv1alpha1.AddToScheme(scheme))

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		// Local/dev fallback (Docker Desktop kubectl proxy / kubeconfig)
		restCfg, err = restConfigFromKubeconfig()
		if err != nil {
			log.Fatalf("kubernetes config: %v", err)
		}
	}

	k8s, err := client.New(restCfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("kubernetes client: %v", err)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           newRouter(cfg, k8s),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("promo-api listening on :%d (namespace=%s rig=%s webhook=%s)",
			cfg.Port, cfg.Namespace, cfg.RigName, cfg.WebhookURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

type config struct {
	Port          int
	Namespace     string
	RigName       string
	MadEyeProxyURL string
	AgentImage    string
	WebhookURL    string
	GenAxisAPIKey string
	PromoAPIURL   string
}

func loadConfig() config {
	return config{
		Port:           envInt("PORT", 8080),
		Namespace:      envOr("NAMESPACE", "gastown-system"),
		RigName:        DefaultRigName,
		MadEyeProxyURL: envOr("MADEYE_PROXY_URL", DefaultMadEyeProxyURL),
		AgentImage:     envOr("AGENT_IMAGE", "polecat-agent:local"),
		WebhookURL:     envOr("GENAXIS_WEBHOOK_URL", genAxisWebhookURL()),
		GenAxisAPIKey:  os.Getenv("GENAXIS_API_KEY"),
		PromoAPIURL:    strings.TrimRight(envOr("PROMO_API_URL", DefaultPromoAPIURL), "/"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func newRouter(cfg config, k8s client.Client) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth(cfg, k8s))
	mux.HandleFunc("GET /healthz", handleHealth(cfg, k8s))
	mux.HandleFunc("POST /v1/promo/generate", handleGenerate(cfg, k8s))
	mux.HandleFunc("GET /v1/promo/jobs/{name}", handleJobStatus(cfg, k8s))
	mux.HandleFunc("POST /v1/promo/jobs/{name}/upload", handleJobUpload(cfg, k8s))
	mux.HandleFunc("POST /v1/promo/jobs/{name}/webhook", handleJobWebhook(cfg, k8s))
	mux.HandleFunc("POST /v1/promo/jobs/{name}/complete", handleJobComplete(cfg, k8s))
	return mux
}

type healthResponse struct {
	Status   string            `json:"status"`
	Checks   map[string]string `json:"checks"`
	Ready    bool              `json:"ready"`
	TimeUTC  string            `json:"time"`
}

func handleHealth(cfg config, k8s client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		checks := map[string]string{}
		ready := true

		// Kubernetes API + Rig exists
		var rig gastownv1alpha1.Rig
		if err := k8s.Get(ctx, types.NamespacedName{Name: cfg.RigName}, &rig); err != nil {
			checks["rig"] = "error: " + err.Error()
			ready = false
		} else {
			checks["rig"] = string(rig.Status.Phase)
			if rig.Status.Phase != gastownv1alpha1.RigPhaseReady && rig.Status.Phase != "" {
				// Still acceptable if Ready condition is true
				for _, c := range rig.Status.Conditions {
					if c.Type == "Ready" && c.Status == metav1.ConditionTrue {
						checks["rig"] = "Ready"
						break
					}
				}
			}
		}

		// Required secrets (git-creds not needed — Polecats use skipGitInit)
		for _, name := range []string{"genaxis-auth"} {
			var sec corev1.Secret
			if err := k8s.Get(ctx, types.NamespacedName{Namespace: cfg.Namespace, Name: name}, &sec); err != nil {
				checks["secret:"+name] = "missing"
				ready = false
			} else {
				checks["secret:"+name] = "ok"
			}
		}

		// MadEye proxy health
		proxyURL := strings.TrimRight(cfg.MadEyeProxyURL, "/") + "/healthz"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			checks["madeye-proxy"] = "unreachable: " + err.Error()
			ready = false
		} else {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				checks["madeye-proxy"] = "ok"
			} else {
				checks["madeye-proxy"] = fmt.Sprintf("http_%d", resp.StatusCode)
				ready = false
			}
		}

		status := "ok"
		code := http.StatusOK
		if !ready {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, healthResponse{
			Status:  status,
			Checks:  checks,
			Ready:   ready,
			TimeUTC: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

type generateRequest struct {
	Prompt          string   `json:"prompt"`
	RequestID       string   `json:"request_id"`
	Name            string   `json:"name,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	SourceDocuments []string `json:"source_documents,omitempty"`
}

type generateResponse struct {
	JobName   string `json:"job_name"`
	RequestID string `json:"request_id"`
	Namespace string `json:"namespace"`
	Rig       string `json:"rig"`
	StatusURL string `json:"status_url"`
	Message   string `json:"message"`
}

func handleGenerate(cfg config, k8s client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		prompt := strings.TrimSpace(req.Prompt)
		if prompt == "" {
			writeErr(w, http.StatusBadRequest, "prompt is required")
			return
		}
		if len(prompt) > 4000 {
			writeErr(w, http.StatusBadRequest, "prompt too long (max 4000 chars)")
			return
		}
		requestID := strings.TrimSpace(req.RequestID)
		if requestID == "" {
			writeErr(w, http.StatusBadRequest, "request_id is required")
			return
		}
		if len(requestID) > 128 {
			writeErr(w, http.StatusBadRequest, "request_id too long (max 128 chars)")
			return
		}

		name := sanitizeName(req.Name)
		if name == "" {
			name = "promo-" + shortID()
		}

		sourceDocs, err := normalizeSourceDocuments(req.SourceDocuments)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		deadline := int64(3600)

		task := fmt.Sprintf(`You are generating promo scripts. The promo pipeline is at a fixed workspace path (no git clone).

PIPELINE ROOT (cd here first):
%s

USER REQUEST:
%s

JOB NAME: %s
REQUEST ID: %s

INSTRUCTIONS:
1. Your FIRST action must be a tool call (Read CLAUDE.md at the pipeline root). Do not reply with planning-only text.
2. cd to the pipeline root above and follow CLAUDE.md / workflows (W2→W3→W4).
3. Write outputs under working files/ relative to the pipeline root.
4. Do NOT git add, commit, push, or create a pull request. Leave git untouched.
5. Do NOT call any HTTP API yourself.
6. When W4 is complete, run promo-finish — see scripts/PROMO_FINISH.md in the pipeline root:

   promo-finish finish map=<path> briefs=<path> script=<path> receipt=<path>

   Use paths relative to the pipeline root or absolute paths. Stop when you see PROMO_FINISH_OK on stderr.
%s`, DefaultPromoWorkspacePath, prompt, name, requestID, sourceDocumentsTaskBlock(name, len(sourceDocs)))

		agentEnv := []corev1.EnvVar{
			{Name: "ANTHROPIC_MODEL", Value: "claude-opus-4-8"},
			{Name: "ANTHROPIC_DEFAULT_OPUS_MODEL", Value: "claude-opus-4-8"},
			{Name: "ANTHROPIC_DEFAULT_SONNET_MODEL", Value: "claude-opus-4-8"},
			{Name: "ANTHROPIC_DEFAULT_HAIKU_MODEL", Value: "claude-opus-4-8"},
			{Name: "ANTHROPIC_AUTH_TOKEN", Value: DefaultProxyMasterKey},
			{Name: "PROMO_API_URL", Value: cfg.PromoAPIURL},
			{Name: "PROMO_JOB_NAME", Value: name},
			{Name: "GENAXIS_REQUEST_ID", Value: requestID},
			{Name: "PROMO_WORKSPACE_PATH", Value: DefaultPromoWorkspacePath},
		}
		annotations := map[string]string{
			annotationRequestID: requestID,
		}
		if len(sourceDocs) > 0 {
			raw, err := sourceDocumentsAnnotation(sourceDocs)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "source_documents annotation: "+err.Error())
				return
			}
			annotations[AnnotationSourceDocuments] = raw
		}
		// LOCAL TEST ONLY — remove before push (see DefaultDevOutputHostPath in defaults.go).
		if DefaultDevOutputHostPath != "" {
			annotations[AnnotationDevOutputHost] = DefaultDevOutputHostPath
			agentEnv = append(agentEnv, corev1.EnvVar{Name: "PROMO_DEV_OUTPUT_DIR", Value: DefaultDevOutputMountPath})
		}

		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: cfg.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/name": "promo-api",
					"gastown.io/flow":        "promo-script",
					"gastown.io/rig":         cfg.RigName,
					"gastown.io/request-id":  sanitizeLabel(requestID),
				},
				Annotations: annotations,
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:             cfg.RigName,
				DesiredState:    gastownv1alpha1.PolecatDesiredWorking,
				BeadID:          "pst-" + name,
				TaskDescription: task,
				ExecutionMode:   gastownv1alpha1.ExecutionModeKubernetes,
				Agent:           gastownv1alpha1.AgentTypeClaudeCode,
				AgentConfig: &gastownv1alpha1.AgentConfig{
					Provider: gastownv1alpha1.LLMProviderLiteLLM,
					Model:    "claude-opus-4-8",
					ModelProvider: &gastownv1alpha1.ModelProviderConfig{
						Endpoint: cfg.MadEyeProxyURL,
					},
					// Force every Claude Code model tier onto MadEye opus-only access.
					Env: agentEnv,
				},
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					SkipGitInit:           true,
					WorkspacePath:         DefaultPromoWorkspacePath,
					Image:                 cfg.AgentImage,
					ActiveDeadlineSeconds: &deadline,
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("1"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					},
				},
			},
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		if err := k8s.Create(ctx, polecat); err != nil {
			if apierrors.IsAlreadyExists(err) {
				writeErr(w, http.StatusConflict, "job name already exists: "+name)
				return
			}
			writeErr(w, http.StatusInternalServerError, "failed to create polecat: "+err.Error())
			return
		}

		// Create can strip skipGitInit/workspacePath (CRD defaulting); merge-patch persists them.
		patchData := fmt.Sprintf(
			`{"spec":{"kubernetes":{"skipGitInit":true,"workspacePath":%q}}}`,
			DefaultPromoWorkspacePath,
		)
		if err := k8s.Patch(ctx, polecat, client.RawPatch(types.MergePatchType, []byte(patchData))); err != nil {
			log.Printf("warn: polecat %s created but skipGitInit patch failed: %v", name, err)
		}

		writeJSON(w, http.StatusAccepted, generateResponse{
			JobName:   name,
			RequestID: requestID,
			Namespace: cfg.Namespace,
			Rig:       cfg.RigName,
			StatusURL: fmt.Sprintf("/v1/promo/jobs/%s", name),
			Message:   "promo script generation started",
		})

		go postGenAxisWebhook(cfg, GenAxisTypePromoStarted, requestID, map[string]any{
			"job_name":  name,
			"status":    "started",
			"prompt":    prompt,
			"namespace": cfg.Namespace,
			"rig":       cfg.RigName,
		}, "")
	}
}

type jobStatusResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Phase     string `json:"phase"`
	PodName   string `json:"pod_name,omitempty"`
	PodActive bool   `json:"pod_active"`
	BeadID    string `json:"bead_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

func handleJobStatus(cfg config, k8s client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			writeErr(w, http.StatusBadRequest, "job name required")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		var polecat gastownv1alpha1.Polecat
		if err := k8s.Get(ctx, types.NamespacedName{Namespace: cfg.Namespace, Name: name}, &polecat); err != nil {
			if apierrors.IsNotFound(err) {
				writeErr(w, http.StatusNotFound, "job not found")
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		msg := ""
		for _, c := range polecat.Status.Conditions {
			if c.Status == metav1.ConditionTrue {
				msg = c.Message
				break
			}
		}

		writeJSON(w, http.StatusOK, jobStatusResponse{
			Name:      polecat.Name,
			Namespace: polecat.Namespace,
			Phase:     string(polecat.Status.Phase),
			PodName:   polecat.Status.PodName,
			PodActive: polecat.Status.PodActive,
			BeadID:    polecat.Status.AssignedBead,
			Message:   msg,
		})
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func sanitizeName(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	var b strings.Builder
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// sanitizeLabel makes a value safe for a Kubernetes label (DNS-ish subset).
func sanitizeLabel(in string) string {
	in = strings.ToLower(strings.TrimSpace(in))
	var b strings.Builder
	for _, r := range in {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-_.")
	if out == "" {
		return "unknown"
	}
	if len(out) > 63 {
		out = out[:63]
	}
	return out
}

func shortID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
