package databases

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var redisClient *redis.Client
var mongoClient *mongo.Client

func GetRedis() *redis.Client {
	if redisClient == nil {
		fmt.Println("[Databases] Connecting to RedisDB")
		opt, err := redis.ParseURL(os.Getenv("REDIS_URI"))

		if err != nil {
			panic(err)
		}
		redisClient = redis.NewClient(opt)

		// Ping the Redis server to check the connection
		_, err = redisClient.Ping(context.Background()).Result()
		if err != nil {
			panic(err)
		}

		fmt.Println("[Databases] Connected to RedisDB")
	}

	return redisClient
}

func GetMongoDatabase() *mongo.Database {
	if mongoClient == nil {
		GetMongo()
	}
	return mongoClient.Database(os.Getenv("MONGO_DB"))
}

func GetMongo() *mongo.Client {
	if mongoClient == nil {
		fmt.Println("[Databases] Connecting to MongoDB")
		clientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URI"))

		// Connect to MongoDB
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := mongo.Connect(ctx, clientOptions)
		if err != nil {
			panic(err)
		}

		err = client.Ping(ctx, nil)
		if err != nil {
			panic(err)
		}

		mongoClient = client
		fmt.Println("[Databases] Connected to MongoDB")
	}

	return mongoClient
}
