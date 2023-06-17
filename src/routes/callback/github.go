package callbackRoutes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/x1xo/Auth/src/databases"
	"github.com/x1xo/Auth/src/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type GithubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Error       string `json:"error"`
}

func CallbackGithub(c *fiber.Ctx) error {
	state := c.Query("state", "")
	code := c.Query("code", "")

	result, err := databases.GetRedis().Get(context.Background(), state).Result()
	if err != nil {
		log.Println(err)
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_STATE",
				"message": "Invalid state was recived on callback. XSS?",
			},
		})
	}
	if result == "" || result != "github" {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_STATE",
				"message": "Invalid state was recived on callback. XSS?",
			},
		})
	}

	databases.GetRedis().Del(context.Background(), state)

	response, err := getGithubResponse(code)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	var userInfo *databases.GithubUser
	var userEmails []*databases.GithubUserEmail

	var wg sync.WaitGroup

	wg.Add(2)
	//Fetch the user info from github
	go func() {
		defer wg.Done()
		u, err := getGithubUserInfo(response)
		if err != nil {
			fmt.Println("[Error] Error while getting user info from github api.", err)
			return
		}
		userInfo = u
	}()

	//Fetch the email list from github
	go func() {
		defer wg.Done()
		e, err := getUserEmail(response)
		if err != nil {
			fmt.Println("[Error] Error while getting user emails from github api.", err)
			return
		}
		userEmails = e
	}()
	wg.Wait()

	userEmail := findPrimaryEmail(userEmails)

	db := databases.GetMongoDatabase()

	var user databases.UserInfo
	err = db.Collection("users").FindOne(context.Background(), bson.M{"email": userEmail.Email}).Decode(&user)
	if err != nil && err == mongo.ErrNoDocuments {
		user = databases.UserInfo{
			Id:        uuid.New().String(),
			Email:     userEmail.Email,
			Username:  userInfo.Username,
			AvatarURL: userInfo.AvatarURL,
			Github:    *userInfo,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		db.Collection("users").InsertOne(context.Background(), &user)
	}

	user.Github = *userInfo
	user.UpdatedAt = time.Now()

	go func() { db.Collection("users").ReplaceOne(context.Background(), bson.M{"id": user.Id}, user) }()

	duration, err := time.ParseDuration(os.Getenv("SESSION_DURATION"))
	if err != nil {
		duration = (time.Hour * 24) * 7
	}

	session, err := utils.CreateSession(user.Id, string(c.Context().UserAgent()), c.IP(), "github", duration)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
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

	c.Set("Authorization", "Bearer "+session.Token)

	return c.Redirect(os.Getenv("REDIRECT_URL"))
}

// getGithubResponse exchanges the code for an access token
//
// code: The auth code the github returend upon successfull oAuth
//
// returns *GithubAccessTokenResponse or an error
func getGithubResponse(code string) (*GithubAccessTokenResponse, error) {
	body := url.Values{
		"client_id":     {os.Getenv("GITHUB_CLIENT_ID")},
		"client_secret": {os.Getenv("GITHUB_CLIENT_SECRET")},
		"code":          {code},
	}.Encode()

	request, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer([]byte(body)))

	if err != nil {
		log.Println("[Error] Couldn't create request for github callback.", err)
		return nil, err
	}

	request.Header.Set("Content-Type", "multipart/form-data")
	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)

	if err != nil {
		log.Println("[Error] Couldn't make request for github callback.", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		log.Println("[Error] Response status from github callback is not 200.")
		return nil, errors.New("response status from github callback is not 200")
	}
	// Read the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var responseMap GithubAccessTokenResponse
	err = json.Unmarshal(responseBody, &responseMap)
	if err != nil {
		log.Println("[Error] Couldn't unmarshal json body for github callback.", err)
		return nil, err
	}

	if responseMap.Error != "" {
		log.Println("[Error] Invalid code was passed to the request somehow.")
		return nil, errors.New("invalid code was passed to the request somehow")
	}

	return &responseMap, nil
}

// getUserEmail returns the user email list from github
//
// data: The response from github (access_token)
//
// returns []*databases.GithubUserEmail or an error
func getUserEmail(data *GithubAccessTokenResponse) ([]*databases.GithubUserEmail, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", data.TokenType+" "+data.AccessToken)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var userEmails []*databases.GithubUserEmail
	err = json.Unmarshal(body, &userEmails)
	if err != nil {
		return nil, err
	}
	return userEmails, nil
}

// getGithubUserInfo returns the user info from github
//
// data: The response from github (access_token)
//
// returns *databases.GithubUser or an error
func getGithubUserInfo(data *GithubAccessTokenResponse) (*databases.GithubUser, error) {

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", data.TokenType+" "+data.AccessToken)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	var userInfo databases.GithubUser
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}

// findPrimaryEmail returns the primary email of github account
//
// userEmails: []*databases.GithubUserEmail - slice of all emails
//
// returns *databases.GithubUserEmail or nil
func findPrimaryEmail(userEmails []*databases.GithubUserEmail) *databases.GithubUserEmail {
	for _, email := range userEmails {
		if email.Primary {
			return email
		}
	}
	return nil
}
