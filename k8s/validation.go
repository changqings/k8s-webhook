package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	v1 "k8s.io/api/admissionregistration/v1"
	k8s_error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func ValidatingPod() http.Handler {
	return &admission.Webhook{
		Handler: admission.HandlerFunc(
			func(ctx context.Context, req admission.Request) admission.Response {
				if req.Namespace == "default" && req.Operation == "delete" {
					return admission.ValidationResponse(false, "not allow by webhook")
				}
				return admission.ValidationResponse(true, "ok, you can do it")
			},
		),
	}
}

func CreateValidatingWebhook(k8sClient *kubernetes.Clientset) error {

	validatingServiceName := strings.Split(webhookServiceName, ".")[0]
	caCrt, err := getCaBundle(k8sClient, webhookSecretName, webhookNamespace)
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
				Name: webhookServiceName,
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubernetes.io/metadata": "default",
					},
				},
				Rules: []v1.RuleWithOperations{
					{
						Operations: []v1.OperationType{
							v1.Delete,
						},
						Rule: v1.Rule{
							APIGroups:   []string{"core"},
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
			slog.Info("ValidatingWebhookConfiguration already exists, %s.%s", valid_webhook.Name, valid_webhook.Namespace)
			return nil
		} else {
			return err
		}
	}

	return nil
}

func getCaBundle(k8sClient *kubernetes.Clientset, secretName, secretNamespace string) ([]byte, error) {

	s, err := k8sClient.CoreV1().Secrets(secretNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	d, ok := s.Data["ca.crt"]
	if !ok {
		return nil, fmt.Errorf("ca.crt not found in secret %s/%s.Data", secretName, secretNamespace)
	}
	return d, nil
}
