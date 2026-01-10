package main

import (
	"cognitive-server/internal/app"
	"log"
)

func main() {
	application, appErr := app.New()
	if appErr != nil {
		log.Fatalf("App init failed: %v", appErr)
	}

	if runErr := application.Run(); runErr != nil {
		log.Fatalf("App run failed: %v", runErr)
	}
}
