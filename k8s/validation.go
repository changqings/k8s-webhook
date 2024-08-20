package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	admission_v1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admissionregistration/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8s_error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func ValidatingPod(k8sClient *kubernetes.Clientset) http.Handler {
	return &admission.Webhook{
		Handler: admission.HandlerFunc(
			func(ctx context.Context, req admission.Request) admission.Response {
				podCanNotBeDeleted := false

				podName := req.Name
				podNamespace := req.Namespace

				pod, err := k8sClient.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
				if err != nil {
					return admission.ValidationResponse(false, "get pod error")
				}

				if v, ok := pod.Labels["allow-delete"]; ok && v == "false" {
					podCanNotBeDeleted = true
				}

				if req.Operation == admission_v1.Delete && podCanNotBeDeleted {
					slog.Info("pod can not be deleted with labels allow-delete=false", "name", req.Name, "namespace", req.Namespace)
					return admission.ValidationResponse(false, "not allow by webhook")
				}
				return admission.ValidationResponse(true, "ok")
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
				Name: "pod-webhook.some.cn",
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubernetes.io/metadata.name": "default",
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

func CheckCertCrdExits(client *apiextv1.Clientset) bool {
	_, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "certificates.cert-manager.io", metav1.GetOptions{})
	return err == nil
}
