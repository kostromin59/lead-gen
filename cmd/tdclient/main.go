package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/kostromin59/lead-gen/internal/clients/tdclient"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		slog.Warn(".env not found", slog.String("err", err.Error()))
	}

	apiIDRaw := os.Getenv("APP_ID")
	apiID, err := strconv.Atoi(apiIDRaw)
	if err != nil {
		panic(err)
	}

	tdc := tdclient.New("s. kostromin", os.Getenv("PHONE"), os.Getenv("API_HASH"), int32(apiID))
	defer tdc.Close()

	eventAuthProvider := tdclient.NewEventAuthProvider(
		func(ctx context.Context, id string, ch chan<- string) {
			var code string

			fmt.Printf("Enter code for account id %q: ", id)
			fmt.Scanln(&code)
			ch <- code
		},
		func(ctx context.Context, id string, ch chan<- string) {
			var password string

			fmt.Printf("Enter password for account id %q: ", id)
			fmt.Scanln(&password)
			ch <- password

		},
	)

	if err := tdc.Authorize(context.Background(), eventAuthProvider); err != nil {
		panic(err)
	}

	me, err := tdc.Client().GetMe()
	if err != nil {
		panic(err)
	}

	log.Printf("FirstName: %q", me.FirstName)
}
