package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
)

func newApp() *fiber.App {
	app := fiber.New()
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	return app
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(newApp().Listen(":" + port))
}
