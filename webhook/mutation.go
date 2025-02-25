package k8swebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	admission_v1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type MutatePod struct {
}

func NewMutatePod() *MutatePod {
	return &MutatePod{}
}
func (mp *MutatePod) MutateHandler() *admission.Webhook {
	return &admission.Webhook{
		Handler: admission.HandlerFunc(
			func(ctx context.Context, req admission.Request) admission.Response {
				if req.AdmissionRequest.Operation == admission_v1.Create ||
					req.AdmissionRequest.Operation == admission_v1.Update {
					slog.Info("create pod patch labels", "namespace", req.Namespace)

					pod := corev1.Pod{}

					err := decoder.Decode(req, &pod)
					if err != nil {
						return admission.Errored(http.StatusBadRequest, err)
					}

					if pod.Labels == nil {
						pod.Labels = make(map[string]string)
					}
					v, ok := pod.Labels["k8s-webhook"]
					if ok && v == "test" {
						slog.Error("pod labels k8s-webhook=test have exist, skip")
						return admission.Allowed("pod labels k8s-webhook=test have exist, skip")
					}
					pod.Labels["k8s-webhook"] = "test"

					pb, err := json.Marshal(pod)
					if err != nil {
						slog.Error("marshal pod labels error", "error", err)
						return admission.Errored(http.StatusInternalServerError, err)
					}
					return admission.PatchResponseFromRaw(req.Object.Raw, pb)
				}
				return admission.Allowed(fmt.Sprintf("get pod req.Operation = %s,skip mutate", req.AdmissionRequest.Operation))
			},
		),
	}
}

func CreateMutatingWebhook(k8sClient *kubernetes.Clientset, namespaces []string) error {

	mutateServiceName := strings.Split(webhookServiceName, ".")[0]
	caCrt, err := GetCaBundle(k8sClient, webhookSecretName, webhookNamespace)
	if err != nil {
		return err
	}

	mutate_webhook := v1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: webhookNamespace,
			Name:      mutateServiceName,
		},
		Webhooks: []v1.MutatingWebhook{
			{
				Name: "pod-webhook.some.cn",
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "kubernetes.io/metadata.name",
							Operator: "In",
							Values:   namespaces,
						},
					},
				},
				Rules: []v1.RuleWithOperations{
					{
						Operations: []v1.OperationType{
							v1.Create,
						},
						Rule: v1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
				ClientConfig: v1.WebhookClientConfig{
					Service: &v1.ServiceReference{
						Namespace: webhookNamespace,
						Name:      mutateServiceName,
						Path:      &WebhookMutatePath,
						Port:      &TLSPort,
					},
					CABundle: caCrt,
				},
				AdmissionReviewVersions: []string{"v1"},
				SideEffects: func() *v1.SideEffectClass {
					var none v1.SideEffectClass = "None"
					return &none
				}(),
			},
		},
	}

	_, err = k8sClient.AdmissionregistrationV1().MutatingWebhookConfigurations().
		Create(context.Background(), &mutate_webhook, metav1.CreateOptions{})
	if err != nil {
		if k8s_error.IsAlreadyExists(err) {
			slog.Info("mutatingWebhookConfiguration already exists", "name", mutate_webhook.Name, "namespace", mutate_webhook.Namespace)
			return nil
		} else {
			return err
		}
	}
	slog.Info("mutatingWebhookConfiguration create success", "name", mutate_webhook.Name, "namespace", mutate_webhook.Namespace)

	return nil
}
