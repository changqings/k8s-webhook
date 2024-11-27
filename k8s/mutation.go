package k8s

import (
	"context"
	"log/slog"
	"strings"

	"gomodules.xyz/jsonpatch/v2"
	admission_v1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/admissionregistration/v1"
	k8s_error "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func MutatingPod() *admission.Webhook {
	return &admission.Webhook{
		Handler: admission.HandlerFunc(
			func(ctx context.Context, req admission.Request) admission.Response {
				if req.AdmissionRequest.Operation == admission_v1.Create ||
					req.AdmissionRequest.Operation == admission_v1.Update {
					slog.Info("create pod patch labels", "namespace", req.Namespace)
					// patch path should exsits, you should check the patch in admission request
					// for example, if pod has no labels at beginning, the path="/metadata/labels/aa"
					// can not be created. And you should use path="/metadata/labels" and and value=<map[string]string>

					// Also you can get k8s object, and check the path
					// obj := &unstructured.Unstructured{}
					// if err := json.Unmarshal(req.Object.Raw, obj); err != nil {
					// 	return admission.Errored(http.StatusBadRequest, err)
					// }

					// _, found, err := unstructured.NestedFieldCopy(obj.Object, "spec", "dnsConfig")
					// if err != nil {
					// 	return admission.Errored(http.StatusInternalServerError, err)
					// }

					return admission.Patched(
						"add label",
						jsonpatch.JsonPatchOperation{
							Operation: "add",
							Path:      "/metadata/labels/k8s-webhook",
							Value:     "test",
						},
					)
				} else {
					return admission.Allowed("ok")
				}
			},
		),
	}
}

func CreateMutatingWebhook(k8sClient *kubernetes.Clientset) error {

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
							Values: []string{
								"default",
							},
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
