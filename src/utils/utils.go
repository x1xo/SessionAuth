package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/x1xo/Auth/src/databases"
)

// CreateSesssions create new session and saves it to redis
//
// userId - the user's id for the session
// userAgned - the user's user agent
// ipAddress - the user's ip address
// provider - the oAuth provider
//
// returns *databases.UserSession or an error
func CreateSession(userId, userAgent, ipAddress, provider string, expiresAt time.Duration) (*databases.UserSession, error) {

	ipInfo, err := GetIPInfo(ipAddress)
	if err != nil {
		return nil, err
	}

	sessionLength, err := strconv.Atoi(os.Getenv("SESSION_LENGTH"))
	if err != nil {
		sessionLength = 64 //for 256bit (64*4)
	}

	sessionToken, err := RandomId(sessionLength / 2) // length/2 because hex converts one byte to two
	if err != nil {
		return nil, err
	}

	userSession := databases.UserSession{
		Id:        uuid.New().String(),
		Token:     sessionToken,
		UserId:    userId,
		UserAgent: userAgent,
		Provider:  provider,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(expiresAt),
		IPAddress: *ipInfo,
	}

	json, err := json.Marshal(userSession)
	if err != nil {
		return nil, err
	}

	if err := databases.GetRedis().Set(context.Background(), userSession.Token, json, expiresAt).Err(); err != nil {
		return nil, err
	}
	if err := databases.GetRedis().Set(context.Background(), userId+"_"+userSession.Id, userSession.Token, expiresAt).Err(); err != nil {
		return nil, err
	}

	return &userSession, nil
}

// GetIPInfo returns information about the users ip address
//
// ipAddress - the users ip address
//
// returns *models.IPAddressInfo or an error
func GetIPInfo(ipAddress string) (*databases.IPAddressInfo, error) {
	infoReq, err := http.Get("https://ipinfo.io/" + ipAddress + "/json")
	if err != nil {
		return nil, err
	}

	defer infoReq.Body.Close()

	body, err := io.ReadAll(infoReq.Body)
	if err != nil {
		return nil, err
	}

	var ipInfo databases.IPAddressInfo
	err = json.Unmarshal(body, &ipInfo)
	if err != nil {
		return nil, err
	}

	return &ipInfo, nil
}

// RandomId returns a random string with a given length
//
// length - the length of the string
//
// returns string or an error
func RandomId(lenght int) (string, error) {
	randomBytes := make([]byte, lenght)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	randomString := hex.EncodeToString(randomBytes)
	return randomString, nil
}

// GetUserToken returns the user token from fiber.Ctx
func GetUserToken(c *fiber.Ctx) string {
	token := ""
	if c.Get("Authorization") != "" {
		split := strings.Split(c.Get("Authorization"), " ")
		if len(split) == 2 {
			token = split[1]
		}
	}
	if token == "" {
		token = c.Cookies("session", "")
	}
	return token
}
