package main

import (
	"context"
	"k8s-webhook/handler"
	"os"
	"os/signal"
)

func main() {

	ctx, cancel := signal.NotifyContext(context.Background(), os.Kill, os.Interrupt)
	defer cancel()

	// run health check in a goroutine
	go func() {
		err := handler.HealthCheck()
		if err != nil {
			panic(err)
		}
	}()

	// run webhook server
	webhookClient := handler.NewWebHookClient()
	webhookServer, err := webhookClient.GetWebHookServer()
	if err != nil {
		panic(err)
	}
	err = webhookServer.Start(ctx)
	if err != nil {
		panic(err)
	}

}
