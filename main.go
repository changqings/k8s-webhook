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
	"github.com/go-logr/logr"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	log "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func main() {

	log.SetLogger(klog.LoggerWithName(logr.Logger{}, "k8s-webhook"))

	restConfig := k8scrdClient.GetRestConfig()
	k8sC := k8scrdClient.GetClient()
	certC := versioned.NewForConfigOrDie(restConfig)
	crdC := apiextv1.NewForConfigOrDie(restConfig)

	// check crd cert
	if !k8s.CheckCertCrdExits(crdC) {
		panic("crd cert not found, plase install cert-manager first")
	}

	if err := k8s.SetUpCertManager(k8sC, certC); err != nil {
		panic(err)
	}

	//
	if err := k8s.CreateValidatingWebhook(k8sC); err != nil {
		panic(err)
	}
	if err := k8s.CreateMutatingWebhook(k8sC); err != nil {
		panic(err)
	}

	server := webhook.NewServer(webhook.Options{
		CertDir:  filepath.Join(homedir.HomeDir(), k8s.TLSCertDir),
		CertName: k8s.CertName,
		KeyName:  k8s.KeyName,
		Port:     int(k8s.TLSPort)})

	server.Register(k8s.WebhookValidPath, k8s.ValidatingPod(k8sC).WithRecoverPanic(true))
	server.Register(k8s.WebhookMutatePath, k8s.MutatingPod().WithRecoverPanic(true))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	go healthCheck()
	err := server.Start(ctx)
	if err != nil {
		panic(err)
	}

}

func healthCheck() {
	http.Handle("/health_check", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		}))
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
