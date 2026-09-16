package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"plaiflow/api/internal/line"
)

func main() {
	image, err := os.ReadFile(os.Getenv("RICH_MENU_IMAGE"))
	if err != nil {
		log.Fatal("read RICH_MENU_IMAGE: ", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id, err := (line.RichMenuPublisher{}).Publish(ctx, os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"), os.Getenv("WEB_BASE_URL"), image)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(id)
}
