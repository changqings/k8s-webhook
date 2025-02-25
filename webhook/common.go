package k8swebhook

import (
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	decoder = admission.NewDecoder(scheme.Scheme)
)
