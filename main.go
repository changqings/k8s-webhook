package main

import (
	"context"
	"flag"
	"k8s-webhook/handler"
	"log"
	"os/signal"
	"strings"
	"syscall"
)

var (
	applyNamespace string
)

func main() {

	flag.StringVar(&applyNamespace, "namespaces", "default", "namespace to apply the webhook,if have many namespace,use comma to split, like: default,kube-system")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// run health check in a goroutine
	go func() {
		err := handler.HealthCheck()
		if err != nil {
			log.Fatalf("main.healthCheck err=%v\n", err)
		}
	}()

	applyNamespaces := strings.Split(applyNamespace, ",")
	// run webhook server
	webhookClient, err := handler.NewWebHookClient(applyNamespaces)
	if err != nil {
		log.Fatalf("main.NewWebHookClient err=%v\n", err)
	}

	webhookServer, err := webhookClient.GetWebHookServer()
	if err != nil {
		log.Fatalf("main.GetWebHookServer err=%v\n", err)
	}
	err = webhookServer.Start(ctx)
	if err != nil {
		log.Fatalf("main.webhookServer.Start err=%v\n", err)
	}

}
