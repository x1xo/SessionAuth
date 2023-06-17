package callbackRoutes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/x1xo/Auth/src/databases"
	"github.com/x1xo/Auth/src/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DiscordAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
}

func CallbackDiscord(c *fiber.Ctx) error {
	state := c.Query("state", "")
	code := c.Query("code", "")

	result, err := databases.GetRedis().Get(context.Background(), state).Result()
	if err != nil {
		log.Println("[Error] Couldn't get state from redis: \n", err)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INVALID_STATE",
				"message": "Invalid state was recived on callback. XSS?",
			},
		})
	}
	if result == "" || result != "discord" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INVALID_STATE",
				"message": "Invalid state was recived on callback. XSS?",
			},
		})
	}

	databases.GetRedis().Del(context.Background(), state)

	response, err := getDiscordResponse(code)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	if response.Error != "" {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	userInfo, err := getDiscordUserInfo(response)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	db := databases.GetMongoDatabase()

	var user databases.UserInfo
	err = db.Collection("users").FindOne(context.Background(), bson.M{"email": userInfo.Email}).Decode(&user)
	if err != nil && err == mongo.ErrNoDocuments {
		user = databases.UserInfo{
			Id:        uuid.New().String(),
			Email:     userInfo.Email,
			Username:  userInfo.Username,
			AvatarURL: userInfo.AvatarURL,
			Discord:   *userInfo,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.Collection("users").InsertOne(context.Background(), &user)
	}

	user.Discord = *userInfo
	user.UpdatedAt = time.Now()

	go func() { db.Collection("users").ReplaceOne(context.Background(), bson.M{"id": user.Id}, user) }()

	duration, err := time.ParseDuration(os.Getenv("SESSION_DURATION"))
	if err != nil {
		duration = (time.Hour * 24) * 7
	}

	session, err := utils.CreateSession(user.Id, string(c.Context().UserAgent()), c.IP(), "discord", duration)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":  "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    session.Token,
		Expires:  time.Now().Add(time.Hour * 3),
		HTTPOnly: true,
		Secure:   os.Getenv("ENVIRONMENT") == "production",
	})

	c.Set("Authorization", "Bearer " + session.Token)

	return c.Redirect(os.Getenv("REDIRECT_URL"))

}
//getDiscordResponse exchanges the code for access token
func getDiscordResponse(code string) (*DiscordAccessTokenResponse, error) {
	body := url.Values{
		"client_id":     {os.Getenv("DISCORD_CLIENT_ID")},
		"client_secret": {os.Getenv("DISCORD_CLIENT_SECRET")},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {os.Getenv("CALLBACK_URL") + "/callback/discord"},
	}.Encode()

	request, err := http.NewRequest("POST", "https://discord.com/api/oauth2/token", bytes.NewBuffer([]byte(body)))

	if err != nil {
		log.Println("[Error] Couldn't create request for discord callback.", err)
		return nil, err
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		log.Println("[Error] Couldn't make request for discord callback.", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Println("[Error] Response status from discord callback is not 200.")
		return nil, errors.New("response status from discord callback is not 200")
	}
	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var responseMap DiscordAccessTokenResponse
	err = json.Unmarshal(responseBody, &responseMap)
	if err != nil {
		log.Println("[Error] Couldn't unmarshal json body for discord callback.", err)
		return nil, err
	}

	if responseMap.Error != "" {
		log.Println("[Error] Invalid code was passed to the request somehow.")
		return nil, errors.New("invalid code was passed to the request somehow")
	}

	return &responseMap, nil
}
//getDiscordUserInfo gets the user info from discord api
func getDiscordUserInfo(data *DiscordAccessTokenResponse) (*databases.DiscordUser, error) {
	request, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		log.Println("[Error] Couldn't create request for discord callback.", err)
		return nil, err
	}

	request.Header.Set("Authorization", data.TokenType+" "+data.AccessToken)
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Println("[Error] Couldn't make request for discord callback.", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Println("[Error] Response status from discord callback is not 200.")
		return nil, err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var userInfo databases.DiscordUser
	err = json.Unmarshal(responseBody, &userInfo)
	if err != nil {
		return nil, err
	}
	userInfo.AvatarURL = "https://cdn.discordapp.com/avatars/" + userInfo.Id + "/" + userInfo.Avatar + ".png"
	userInfo.Avatar = ""

	return &userInfo, nil
}
