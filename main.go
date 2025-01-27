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

	go func(ctx context.Context) {
		err := handler.HealthCheck(ctx)
		if err != nil {
			panic(err)
		}
	}(ctx)

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
