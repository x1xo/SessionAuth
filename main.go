package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
	"github.com/x1xo/Auth/src/databases"
	"github.com/x1xo/Auth/src/routes"
	callbackRoutes "github.com/x1xo/Auth/src/routes/callback"
)

func main() {
	godotenv.Load()
	go databases.GetRedis()
	databases.GetMongo()

	app := fiber.New(fiber.Config{
		ProxyHeader:             "X-Forwarded-For",
		EnableTrustedProxyCheck: false,
	})
	app.Use(logger.New())

	app.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin,Content-Type,Accept,Content-Length,Accept-Language,Accept-Encoding,Connection,Access-Control-Allow-Origin",
		AllowCredentials: true,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowOrigins:     os.Getenv("ALLOWED_ORIGINS"),
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Identity provider by x1xo. All rights reserved.")
	})

	app.Get("/api/user", routes.GetUser)
	app.Get("/api/user/sessions", routes.GetUserSessions)
	app.Delete("/api/user/sessions/invalidate_all", routes.InvalidateAllSessions)
	app.Delete("/api/user/sessions/:sessionId", routes.InvalidateSession)

	app.Get("/login", routes.Login)

	app.Get("/callback/github", callbackRoutes.CallbackGithub)
	app.Get("/callback/discord", callbackRoutes.CallbackDiscord)
	app.Get("/callback/google", callbackRoutes.CallbackGoogle)

	environment := os.Getenv("ENVIRONMENT")
	port := os.Getenv("PORT")

	if port == "" {
		port = "3000"
	}
	if environment == "" {
		environment = "production"
	}

	if environment == "production" {
		log.Fatal(app.Listen(fmt.Sprintf("%s:%s", "0.0.0.0", port)))
	} else {
		log.Fatal(app.Listen(fmt.Sprintf("%s:%s", "127.0.0.1", port)))
	}
}
