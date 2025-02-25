package handler

import (
	"errors"
	k8swebhook "k8s-webhook/webhook"
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

type Client struct {
	K8sClient  *kubernetes.Clientset
	CertClient *versioned.Clientset
	CrdClient  *apiextv1.Clientset

	Namespace []string
}

func NewWebHookClient(namespaces []string) (*Client, error) {
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

	return &Client{
		K8sClient:  k8sC,
		CertClient: certC,
		CrdClient:  crdC,
		Namespace:  namespaces,
	}, nil
}

func (c *Client) GetWebHookServer() (webhook.Server, error) {

	log.SetLogger(klog.LoggerWithName(logr.Logger{}, "k8s-webhook"))

	// new webhook server
	server := webhook.NewServer(webhook.Options{
		CertDir:  filepath.Join(homedir.HomeDir(), k8swebhook.TLSCertDir),
		CertName: k8swebhook.CertName,
		KeyName:  k8swebhook.KeyName,
		Port:     int(k8swebhook.TLSPort)})

	// check crd cert
	if !k8swebhook.CheckCertCrdExits(c.CrdClient) {
		return nil, errors.New("cert-manager.io crd  not found, plase install cert-manager first")
	}

	err := k8swebhook.SetUpCertManager(c.K8sClient, c.CertClient)
	if err != nil {
		return nil, err
	}

	// validate webhook
	err = k8swebhook.CreateValidatingWebhook(c.K8sClient, c.Namespace)
	if err != nil {
		return nil, err
	}

	valitePod := k8swebhook.NewValitePod()
	server.Register(k8swebhook.WebhookValidPath, valitePod.ValiteHandler().WithRecoverPanic(true))

	// mutate webhook
	err = k8swebhook.CreateMutatingWebhook(c.K8sClient, c.Namespace)
	if err != nil {
		return nil, err
	}
	mutatePod := k8swebhook.NewMutatePod()
	server.Register(k8swebhook.WebhookMutatePath, mutatePod.MutateHandler().WithRecoverPanic(true))

	return server, nil

}
