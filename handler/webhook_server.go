package handler

import (
	"errors"
	"k8s-webhook/k8s"
	"path/filepath"

	"github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	k8scrdClient "github.com/changqings/k8scrd/client"
	"github.com/go-logr/logr"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type WebhookClient struct {
	K8sClient  *kubernetes.Clientset
	CertClient *versioned.Clientset
	CrdClient  *apiextv1.Clientset
}

func NewWebHookClient() (*WebhookClient, error) {
	kubeconfig := k8scrdClient.GetKubeConfig()

	restConfig, err := k8scrdClient.GetRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	k8sC, err := k8scrdClient.GetClient(restConfig)
	if err != nil {
		return nil, err
	}

	certC := versioned.NewForConfigOrDie(restConfig)
	crdC := apiextv1.NewForConfigOrDie(restConfig)

	return &WebhookClient{
		K8sClient:  k8sC,
		CertClient: certC,
		CrdClient:  crdC,
	}, nil
}

func (wc *WebhookClient) GetWebHookServer() (webhook.Server, error) {

	log.SetLogger(klog.LoggerWithName(logr.Logger{}, "k8s-webhook"))

	// new webhook server
	server := webhook.NewServer(webhook.Options{
		CertDir:  filepath.Join(homedir.HomeDir(), k8s.TLSCertDir),
		CertName: k8s.CertName,
		KeyName:  k8s.KeyName,
		Port:     int(k8s.TLSPort)})

	// check crd cert
	if !k8s.CheckCertCrdExits(wc.CrdClient) {
		return nil, errors.New("cert-manager.io crd  not found, plase install cert-manager first")
	}

	err := k8s.SetUpCertManager(wc.K8sClient, wc.CertClient)
	if err != nil {
		return nil, err
	}

	// validate webhook
	err = k8s.CreateValidatingWebhook(wc.K8sClient)
	if err != nil {
		return nil, err
	}

	valitePod := k8s.NewValitePod()
	server.Register(k8s.WebhookValidPath, valitePod.ValiteHandler().WithRecoverPanic(true))

	// mutate webhook
	err = k8s.CreateMutatingWebhook(wc.K8sClient)
	if err != nil {
		return nil, err
	}
	mutatePod := k8s.NewMutatePod()
	server.Register(k8s.WebhookMutatePath, mutatePod.MutateHandler().WithRecoverPanic(true))

	return server, nil

}
