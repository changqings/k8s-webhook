package k8swebhook

import (
	"context"
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

type ValitePod struct{}

func NewValitePod() *ValitePod {
	return &ValitePod{}
}

func (vp *ValitePod) ValiteHandler() *admission.Webhook {
	return &admission.Webhook{
		Handler: admission.HandlerFunc(
			func(ctx context.Context, req admission.Request) admission.Response {
				if req.AdmissionRequest.Operation == admission_v1.Delete {
					pod := corev1.Pod{}

					err := decoder.DecodeRaw(req.Object, &pod)
					if err != nil {
						slog.Error("decode pod error", "error", err)
						return admission.Errored(http.StatusBadRequest, err)
					}

					v, ok := pod.Labels["allow-delete"]
					if ok && v == "false" {
						return admission.ValidationResponse(false, "not allow by webhook")
					}

					return admission.ValidationResponse(true, "pod not have label allow-delete=true, can be deleted")
				}
				return admission.ValidationResponse(true, fmt.Sprintf("get pod req.Operation = %s, skip validate", req.AdmissionRequest.Operation))
			},
		),
	}
}

func CreateValidatingWebhook(k8sClient *kubernetes.Clientset, namespaces []string) error {

	validatingServiceName := strings.Split(webhookServiceName, ".")[0]
	caCrt, err := GetCaBundle(k8sClient, webhookSecretName, webhookNamespace)
	if err != nil {
		return err
	}

	valid_webhook := v1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: webhookNamespace,
			Name:      validatingServiceName,
		},
		Webhooks: []v1.ValidatingWebhook{
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
							v1.Delete,
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
						Name:      validatingServiceName,
						Path:      &WebhookValidPath,
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

	_, err = k8sClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().
		Create(context.Background(), &valid_webhook, metav1.CreateOptions{})
	if err != nil {
		if k8s_error.IsAlreadyExists(err) {
			slog.Info("validatingWebhookConfiguration already exists", "name", valid_webhook.Name, "namespace", valid_webhook.Namespace)
			return nil
		} else {
			return err
		}
	}
	slog.Info("validatingWebhookConfiguration create success", "name", valid_webhook.Name, "namespace", valid_webhook.Namespace)

	return nil
}
