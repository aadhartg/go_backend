package config

import (
	"context"
	"fmt"
	"strconv"

	redis "github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func InitRedis() *redis.Client {
	db, _ := strconv.Atoi(AppConfig.RedisDB)

	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", AppConfig.RedisHost, AppConfig.RedisPort),
		Password: AppConfig.RedisPassword,
		DB:       db,
	})
}

var RedisClient *redis.Client
