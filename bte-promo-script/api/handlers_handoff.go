package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gastownv1alpha1 "github.com/org/gastown-operator/api/v1alpha1"
)

type handoffBody struct {
	RequestID string        `json:"request_id"`
	FilePath  string        `json:"file_path,omitempty"`
	Script    string        `json:"script,omitempty"`
	Files     []filePayload `json:"files,omitempty"`
}

type uploadResponse struct {
	JobName    string     `json:"job_name"`
	RequestID  string     `json:"request_id"`
	Artifacts  []artifact `json:"artifacts"`
}

type webhookHandoffBody struct {
	RequestID string     `json:"request_id"`
	Artifacts []artifact `json:"artifacts"`
}

type webhookResponse struct {
	JobName   string `json:"job_name"`
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
}

func handleJobUpload(cfg config, k8s client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			writeErr(w, http.StatusBadRequest, "job name required")
			return
		}

		var req handoffBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		files, err := normalizeFiles(req)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		_, requestID, err := loadJobRequestID(ctx, k8s, cfg.Namespace, name, req.RequestID)
		if err != nil {
			writeHandoffErr(w, err)
			return
		}

		log.Printf("upload received: job_name=%q request_id=%q files=%d", name, requestID, len(files))
		for _, f := range files {
			log.Printf("  upload input: key=%q path=%q", f.Key, f.FilePath)
		}

		artifacts, err := uploadPromoArtifacts(name, requestID, files)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		go saveDevScript(name, files)

		writeJSON(w, http.StatusOK, uploadResponse{
			JobName:   name,
			RequestID: requestID,
			Artifacts: artifacts,
		})
	}
}

func handleJobWebhook(cfg config, k8s client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			writeErr(w, http.StatusBadRequest, "job name required")
			return
		}

		var req webhookHandoffBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		_, requestID, err := loadJobRequestID(ctx, k8s, cfg.Namespace, name, req.RequestID)
		if err != nil {
			writeHandoffErr(w, err)
			return
		}

		if err := validateArtifacts(req.Artifacts); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		log.Printf("handoff: [promo-finish→promo-api] POST /v1/promo/jobs/%s/webhook job=%q request_id=%q artifacts=%d",
			name, name, requestID, len(req.Artifacts))
		for _, a := range req.Artifacts {
			log.Printf("  handoff artifact: key=%q path=%q file_url=%q s3_key=%q", a.Key, a.FilePath, a.FileURL, a.S3Key)
		}

		result := map[string]any{
			"job_name":  name,
			"status":    "completed",
			"artifacts": req.Artifacts,
			"files":     artifactsByKey(req.Artifacts),
			"rig":       cfg.RigName,
		}
		log.Printf("handoff: firing GenAxis webhook type=%s request_id=%q (triggered by promo-finish)", GenAxisTypePromoCompleted, requestID)
		postGenAxisWebhook(cfg, GenAxisTypePromoCompleted, requestID, result, "")

		writeJSON(w, http.StatusOK, webhookResponse{
			JobName:   name,
			RequestID: requestID,
			Message:   "genaxis completed webhook sent",
		})
	}
}

// handleJobComplete runs upload + webhook in one call (testing / convenience).
func handleJobComplete(cfg config, k8s client.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		if name == "" {
			writeErr(w, http.StatusBadRequest, "job name required")
			return
		}

		var req handoffBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		files, err := normalizeFiles(req)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		_, requestID, err := loadJobRequestID(ctx, k8s, cfg.Namespace, name, req.RequestID)
		if err != nil {
			writeHandoffErr(w, err)
			return
		}

		artifacts, err := uploadPromoArtifacts(name, requestID, files)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}

		go saveDevScript(name, files)

		postGenAxisWebhook(cfg, GenAxisTypePromoCompleted, requestID, map[string]any{
			"job_name":  name,
			"status":    "completed",
			"artifacts": artifacts,
			"files":     artifactsByKey(artifacts),
			"rig":       cfg.RigName,
		}, "")

		writeJSON(w, http.StatusOK, uploadResponse{
			JobName:   name,
			RequestID: requestID,
			Artifacts: artifacts,
		})
	}
}

func normalizeFiles(req handoffBody) ([]filePayload, error) {
	if len(req.Files) > 0 {
		for i, f := range req.Files {
			if strings.TrimSpace(f.Key) == "" {
				return nil, fmt.Errorf("files[%d].key is required", i)
			}
			if strings.TrimSpace(f.FilePath) == "" {
				return nil, fmt.Errorf("files[%d].file_path is required", i)
			}
		}
		return req.Files, nil
	}
	if strings.TrimSpace(req.FilePath) != "" {
		return []filePayload{{Key: "script", FilePath: req.FilePath}}, nil
	}
	return nil, fmt.Errorf("files array with key+file_path is required")
}

func validateArtifacts(artifacts []artifact) error {
	if len(artifacts) == 0 {
		return fmt.Errorf("artifacts array is required (from upload step)")
	}
	for i, a := range artifacts {
		if a.Key == "" {
			return fmt.Errorf("artifact[%d] must include key", i)
		}
		if a.FileURL == "" || a.S3Key == "" {
			return fmt.Errorf("artifact[%d] must include file_url and s3_key", i)
		}
	}
	return nil
}

func loadJobRequestID(ctx context.Context, k8s client.Client, namespace, name, reqID string) (*gastownv1alpha1.Polecat, string, error) {
	var polecat gastownv1alpha1.Polecat
	if err := k8s.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &polecat); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, "", handoffError{code: http.StatusNotFound, msg: "job not found"}
		}
		return nil, "", handoffError{code: http.StatusInternalServerError, msg: err.Error()}
	}

	requestID := strings.TrimSpace(reqID)
	if requestID == "" {
		requestID = polecat.Annotations[annotationRequestID]
	}
	if requestID == "" {
		return nil, "", handoffError{code: http.StatusBadRequest, msg: "request_id is required"}
	}
	if expected := polecat.Annotations[annotationRequestID]; expected != "" && requestID != expected {
		return nil, "", handoffError{code: http.StatusBadRequest, msg: "request_id does not match job"}
	}

	return &polecat, requestID, nil
}

type handoffError struct {
	code int
	msg  string
}

func (e handoffError) Error() string { return e.msg }

func writeHandoffErr(w http.ResponseWriter, err error) {
	if he, ok := err.(handoffError); ok {
		writeErr(w, he.code, he.msg)
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}
