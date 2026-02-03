package main

import (
	"context"

	"github.com/Levy-Tal/gintelemetry"
)

func main() {
	ctx := context.Background()

	tel, router, err := gintelemetry.Start(ctx, gintelemetry.Config{
		ServiceName: "testing-example",
		Endpoint:    "localhost:4317",
		LogLevel:    gintelemetry.LevelInfo,
	})
	if err != nil {
		panic(err)
	}
	defer tel.Shutdown(ctx)

	SetupRoutes(router, tel)

	tel.Log().Info(ctx, "server starting", "port", 8080)
	router.Run(":8080")
}
