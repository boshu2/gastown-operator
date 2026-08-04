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
		log.Printf("promo-api listening on :%d (namespace=%s rig=%s)", cfg.Port, cfg.Namespace, cfg.RigName)
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
	Port           int
	Namespace      string
	RigName        string
	GitRepo        string
	GitBranch      string
	GitSecret      string
	LiteLLMURL     string
	LiteLLMAuthSec string
	AgentImage     string
}

func loadConfig() config {
	return config{
		Port:           envInt("PORT", 8080),
		Namespace:      envOr("NAMESPACE", "gastown-system"),
		RigName:        envOr("RIG_NAME", "local-smoke"),
		GitRepo:        envOr("GIT_REPO", "git@github.com:priyanshur01/gastown-static.git"),
		GitBranch:      envOr("GIT_BRANCH", "main"),
		GitSecret:      envOr("GIT_SECRET", "git-creds"),
		LiteLLMURL:     envOr("LITELLM_URL", "http://litellm.gastown-system.svc:4000"),
		LiteLLMAuthSec: envOr("LITELLM_AUTH_SECRET", "litellm-auth"),
		AgentImage:     envOr("AGENT_IMAGE", "polecat-agent:local"),
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

		// Required secrets
		for _, name := range []string{cfg.GitSecret, cfg.LiteLLMAuthSec} {
			var sec corev1.Secret
			if err := k8s.Get(ctx, types.NamespacedName{Namespace: cfg.Namespace, Name: name}, &sec); err != nil {
				checks["secret:"+name] = "missing"
				ready = false
			} else {
				checks["secret:"+name] = "ok"
			}
		}

		// LiteLLM liveliness
		llURL := strings.TrimRight(cfg.LiteLLMURL, "/") + "/health/liveliness"
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, llURL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			checks["litellm"] = "unreachable: " + err.Error()
			ready = false
		} else {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				checks["litellm"] = "ok"
			} else {
				checks["litellm"] = fmt.Sprintf("http_%d", resp.StatusCode)
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
	Prompt string `json:"prompt"`
	Name   string `json:"name,omitempty"`
	Branch string `json:"branch,omitempty"`
}

type generateResponse struct {
	JobName   string `json:"job_name"`
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

		name := sanitizeName(req.Name)
		if name == "" {
			name = "promo-" + shortID()
		}

		branch := req.Branch
		if branch == "" {
			branch = cfg.GitBranch
		}
		workBranch := fmt.Sprintf("feature/%s", name)
		deadline := int64(3600)

		task := fmt.Sprintf(`You are generating promo scripts in this repository.

USER REQUEST:
%s

INSTRUCTIONS:
1. Explore the repo structure and existing promo/script conventions.
2. Generate or update the promo script(s) needed for the request.
3. Keep changes focused; do not refactor unrelated code.
4. After finishing:
   - git add relevant files
   - git commit -m 'feat(%s): promo script generation'
   - git push origin HEAD
   - create a PR if gh is available (gh pr create --fill), otherwise leave the branch pushed
`, prompt, name)

		polecat := &gastownv1alpha1.Polecat{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: cfg.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "promo-api",
					"gastown.io/flow":             "promo-script",
					"gastown.io/rig":              cfg.RigName,
				},
			},
			Spec: gastownv1alpha1.PolecatSpec{
				Rig:             cfg.RigName,
				DesiredState:    gastownv1alpha1.PolecatDesiredWorking,
				BeadID:          "ls-" + name,
				TaskDescription: task,
				ExecutionMode:   gastownv1alpha1.ExecutionModeKubernetes,
				Agent:           gastownv1alpha1.AgentTypeClaudeCode,
				AgentConfig: &gastownv1alpha1.AgentConfig{
					Provider: gastownv1alpha1.LLMProviderLiteLLM,
					Model:    "claude-opus-4-8",
					ModelProvider: &gastownv1alpha1.ModelProviderConfig{
						Endpoint: cfg.LiteLLMURL,
						APIKeySecretRef: &gastownv1alpha1.SecretKeyRef{
							Name: cfg.LiteLLMAuthSec,
							Key:  "master-key",
						},
					},
					// Force every Claude Code model tier onto MadEye opus-only access.
					Env: []corev1.EnvVar{
						{Name: "ANTHROPIC_MODEL", Value: "claude-opus-4-8"},
						{Name: "ANTHROPIC_DEFAULT_OPUS_MODEL", Value: "claude-opus-4-8"},
						{Name: "ANTHROPIC_DEFAULT_SONNET_MODEL", Value: "claude-opus-4-8"},
						{Name: "ANTHROPIC_DEFAULT_HAIKU_MODEL", Value: "claude-opus-4-8"},
					},
				},
				Kubernetes: &gastownv1alpha1.KubernetesSpec{
					GitRepository: cfg.GitRepo,
					GitBranch:     branch,
					WorkBranch:    workBranch,
					GitSecretRef:  gastownv1alpha1.SecretReference{Name: cfg.GitSecret},
					ApiKeySecretRef: &gastownv1alpha1.SecretKeyRef{
						Name: cfg.LiteLLMAuthSec,
						Key:  "master-key",
					},
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

		writeJSON(w, http.StatusAccepted, generateResponse{
			JobName:   name,
			Namespace: cfg.Namespace,
			Rig:       cfg.RigName,
			StatusURL: fmt.Sprintf("/v1/promo/jobs/%s", name),
			Message:   "promo script generation started",
		})
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

func shortID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
