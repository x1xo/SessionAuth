package routes

import (
	"context"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/x1xo/Auth/src/databases"
	"github.com/x1xo/Auth/src/utils"
	"go.mongodb.org/mongo-driver/bson"
)

// GET "/api/user"
func GetUser(c *fiber.Ctx) error {
	token := utils.GetUserToken(c)
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token couldn't be found in header or cookie",
			},
		})
	}

	result, err := databases.GetRedis().Get(context.Background(), token).Result()
	if result == "" || err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token is invalid.",
			},
		})
	}

	var userSession databases.UserSession
	err = json.Unmarshal([]byte(result), &userSession)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	var userInfo databases.UserInfo
	err = databases.GetMongoDatabase().Collection("users").FindOne(context.Background(), bson.M{"id": userSession.UserId}).Decode(&userInfo)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	return c.JSON(userInfo)
}

// GET "/api/user/sessions"
func GetUserSessions(c *fiber.Ctx) error {
	token := utils.GetUserToken(c)
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token couldn't be found in header or cookie",
			},
		})
	}

	result, err := databases.GetRedis().Get(context.Background(), token).Result()
	if result == "" || err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token is invalid.",
			},
		})
	}

	var currentSession databases.UserSession
	err = json.Unmarshal([]byte(result), &currentSession)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	/* var userSessions []databases.UserSession */

	var sessions []string
	//This will return the user session ids
	databases.GetRedis().Keys(context.Background(), currentSession.UserId+"_*").ScanSlice(&sessions)

	var sessionTokens []string
	//This will return the user session tokens
	sessionTokensI, err := databases.GetRedis().MGet(context.Background(), sessions...).Result()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	//Cast the user session tokens to string
	for i := 0; i < len(sessionTokensI); i++ {
		sessionTokens = append(sessionTokens, sessionTokensI[i].(string))
	}
	//Session informations in json format
	sessionsJSON, err := databases.GetRedis().MGet(context.Background(), sessionTokens...).Result()

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	var parsedSessions []databases.UserSession
	//parsing the sessions
	for _, sessionJSON := range sessionsJSON {
		var session databases.UserSession
		err = json.Unmarshal([]byte(sessionJSON.(string)), &session)
		if err != nil {
			continue
		}
		session.Token = ""
		parsedSessions = append(parsedSessions, session)
	}

	return c.JSON(parsedSessions)
}

// DELETE "/api/user/sessions/:sessionId"
func InvalidateSession(c *fiber.Ctx) error {
	sessionId := c.Params("sessionId", "")
	if sessionId == "" || len(sessionId) < 36 {
		return c.Status(400).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "SessionId parameter is invalid.",
			},
		})
	}

	token := utils.GetUserToken(c)
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token couldn't be found in header or cookie",
			},
		})
	}

	result, err := databases.GetRedis().Get(context.Background(), token).Result()
	if result == "" || err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token is invalid.",
			},
		})
	}

	var currentSession databases.UserSession
	err = json.Unmarshal([]byte(result), &currentSession)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	sessionToken, err := databases.GetRedis().Get(context.Background(), currentSession.UserId+"_"+sessionId).Result()
	if sessionToken == "" || err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Session was not found.",
			},
		})
	}

	databases.GetRedis().Del(context.Background(), currentSession.UserId+"_"+sessionId)

	session, err := databases.GetRedis().GetDel(context.Background(), sessionToken).Result()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	var parsedSession databases.UserSession
	err = json.Unmarshal([]byte(session), &parsedSession)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	parsedSession.Token = ""

	return c.Status(200).JSON(fiber.Map{
		"success": true,
	})
}

// DELETE "/api/user/sessions/invalidate_all"

func InvalidateAllSessions(c *fiber.Ctx) error {
	token := utils.GetUserToken(c)
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token couldn't be found in header or cookie",
			},
		})
	}

	result, err := databases.GetRedis().Get(context.Background(), token).Result()
	if result == "" || err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "UNAUTHENTICATED",
				"message": "Session token is invalid.",
			},
		})
	}

	var currentSession databases.UserSession
	err = json.Unmarshal([]byte(result), &currentSession)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	sessionIds, err := databases.GetRedis().Keys(context.Background(), currentSession.UserId+"_*").Result()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	sessionTokens, err := databases.GetRedis().MGet(context.Background(), sessionIds...).Result()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	pipe := databases.GetRedis().Pipeline()
	for i := 0; i < len(sessionTokens); i++ {
		pipe.Del(context.Background(), sessionTokens[i].(string))
	}
	for i := 0; i < len(sessionIds); i++ {
		pipe.Del(context.Background(), sessionIds[i])
	}

	_, err = pipe.Exec(context.Background())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "Something went wrong on our side. Try again later.",
			},
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"success": true,
	})

}
