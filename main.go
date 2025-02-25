package main

import (
	"context"
	"k8s-webhook/handler"
	"log"
	"os/signal"
	"syscall"
)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	// run health check in a goroutine
	go func() {
		err := handler.HealthCheck()
		if err != nil {
			log.Fatalf("main.healthCheck err=%v\n", err)
		}
	}()

	// run webhook server
	webhookClient, err := handler.NewWebHookClient()
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
