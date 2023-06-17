package routes

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/x1xo/Auth/src/databases"
	"github.com/x1xo/Auth/src/utils"
)

var SCOPES = map[string]string{
	"github":  "user%20user:email%20repo%20repo_deployment",
	"google":  "profile+email",
	"discord": "identify%20guilds.join%20email",
}

func getRedirectURL(provider string) (string, error) {
	callbackURL := os.Getenv("CALLBACK_URL")
	if callbackURL == "" {
		return "", errors.New("callback url not found")
	}

	var redirectURL string
	var err error

	switch provider {
	case "github":
		clientID := os.Getenv("GITHUB_CLIENT_ID")
		if clientID == "" {
			err = errors.New("github client id is not set")
			break
		}

		redirectURL = "https://github.com/login/oauth/authorize?scope="
		redirectURL += SCOPES["github"]
		redirectURL += "&redirect_uri=" + callbackURL + "/callback/github"
		redirectURL += "&client_id=" + clientID

		return redirectURL, nil
	case "google":
		clientID := os.Getenv("GOOGLE_CLIENT_ID")
		if clientID == "" {
			err = errors.New("google client id is not set")
			break
		}

		redirectURL = "https://accounts.google.com/o/oauth2/v2/auth?prompt=consent&response_type=code&access_type=offline"
		redirectURL += "&scope=" + SCOPES["google"]
		redirectURL += "&redirect_uri=" + callbackURL + "/callback/google"
		redirectURL += "&client_id=" + clientID
	case "discord":
		clientID := os.Getenv("DISCORD_CLIENT_ID")
		if clientID == "" {
			err = errors.New("discord client id is not set")
			break
		}

		redirectURL = "https://discord.com/oauth2/authorize?response_type=code&prompt=consent&scope="
		redirectURL += SCOPES["discord"]
		redirectURL += "&redirect_uri=" + callbackURL + "/callback/discord"
		redirectURL += "&client_id=" + clientID
	default:
		err = errors.New("provider not found")
	}

	return redirectURL, err

}

func Login(c *fiber.Ctx) error {
	provider := c.Query("provider", "github")

	redirectUri, err := getRedirectURL(provider)
	if err != nil {
		log.Println(err)
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "PROVIDER_NOT_FOUND",
				"message": "Provider was not found.",
			},
		})
	}

	redis := databases.GetRedis()
	state, err := utils.RandomId(8)
	if err != nil {
		log.Println(err)
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	err = redis.Set(context.Background(), state, provider, time.Minute*10).Err()
	if err != nil {
		log.Println(err)
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	return c.Redirect(redirectUri + "&state=" + state)
}
