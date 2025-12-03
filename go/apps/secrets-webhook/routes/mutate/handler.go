package mutate

import (
	"context"
	"encoding/json"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/unkeyed/unkey/go/apps/secrets-webhook/internal/services/mutator"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger  logging.Logger
	Mutator *mutator.Mutator
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/mutate" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	admissionReview, err := zen.BindBody[admissionv1.AdmissionReview](s)
	if err != nil {
		h.Logger.Error("failed to parse admission review", "error", err)
		return s.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse admission review"})
	}

	var pod corev1.Pod
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, &pod); err != nil {
		h.Logger.Error("failed to parse pod", "error", err)
		return h.sendResponse(s, admissionReview.Request.UID, false, "failed to parse pod")
	}

	namespace := admissionReview.Request.Namespace

	h.Logger.Info("received admission request",
		"pod", pod.Name,
		"namespace", namespace,
		"uid", admissionReview.Request.UID,
	)

	result, err := h.Mutator.Mutate(ctx, &pod, namespace)
	if err != nil {
		h.Logger.Error("failed to mutate pod", "pod", pod.Name, "namespace", namespace, "error", err)
		return h.sendResponse(s, admissionReview.Request.UID, false, err.Error())
	}

	if result.Mutated {
		h.Logger.Info("mutated pod", "pod", pod.Name, "namespace", pod.Namespace, "message", result.Message)
		return h.sendResponseWithPatch(s, admissionReview.Request.UID, result.Patch)
	}

	h.Logger.Info("skipped pod mutation", "pod", pod.Name, "namespace", pod.Namespace, "message", result.Message)
	return h.sendResponse(s, admissionReview.Request.UID, true, result.Message)
}

func (h *Handler) sendResponse(s *zen.Session, uid types.UID, allowed bool, message string) error {
	response := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
		Response: &admissionv1.AdmissionResponse{UID: uid, Allowed: allowed},
	}

	if message != "" && !allowed {
		response.Response.Result = &metav1.Status{Message: message}
	}

	return s.JSON(http.StatusOK, response)
}

func (h *Handler) sendResponseWithPatch(s *zen.Session, uid types.UID, patch []byte) error {
	patchType := admissionv1.PatchTypeJSONPatch

	response := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
		Response: &admissionv1.AdmissionResponse{
			UID:       uid,
			Allowed:   true,
			Patch:     patch,
			PatchType: &patchType,
		},
	}

	return s.JSON(http.StatusOK, response)
}
