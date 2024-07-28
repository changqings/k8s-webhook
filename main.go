package main

import (
	"context"
	"k8s-webhook/k8s"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	k8scrdClient "github.com/changqings/k8scrd/client"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func main() {

	restConfig := k8scrdClient.GetRestConfig()
	//
	k8sC := k8scrdClient.GetClient()
	certC := versioned.NewForConfigOrDie(restConfig)

	if err := k8s.SetUpCertManager(k8sC, certC); err != nil {
		panic(err)
	}

	server := webhook.NewServer(webhook.Options{
		CertDir:  filepath.Join(homedir.HomeDir(), k8s.TLSCertDir),
		CertName: k8s.CertName,
		KeyName:  k8s.KeyName,
		Port:     int(k8s.TLSPort)})

	server.Register(k8s.WebhookValidPath, k8s.ValidatingPod())

	server.Register("/health_check", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	err := server.Start(ctx)
	if err != nil {
		panic(err)
	}

}
