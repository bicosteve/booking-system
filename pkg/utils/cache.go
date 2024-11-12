package utils

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/bicosteve/booking-system/pkg/entities"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func ConnectToRedis(ctx context.Context, address, password, port string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         address + ":" + port,
		Password:     password,
		DB:           0,
		PoolSize:     1000,
		PoolTimeout:  time.Second * 5,
		MinIdleConns: 32,
	})

	err := client.Ping(ctx).Err()
	if err != nil {
		MessageLogs.ErrorLog.Printf("Could not connect to redis because of %v\n", err)
		os.Exit(1)
	}

	MessageLogs.InfoLog.Println("Redis pinged successfully")

	return client, nil
}

func CreateUser(user entities.UserPayload, c *redis.Client) error {
	var id = genId()
	var score = convertIdToScore(id)
	_ = score
	userFields := map[string]interface{}{
		"username": user.Email,
		"password": user.Password,
	}

	// Using Sets to store unordered unique values of users
	exist, err := c.SIsMember(ctx, usernameUniqueKey(id), user.Email).Result()
	if err != nil {
		MessageLogs.ErrorLog.Printf("an error of occured because %v\n", err)
		return err

	}

	if exist {
		MessageLogs.ErrorLog.Printf("username already registered")
		return errors.New("username already registered")
	}

	// Using hash HSet key value pairs with userKey to create a user
	_, err = c.HSet(ctx, usersKey(strconv.FormatInt(id, 10)), userFields).Result()
	if err != nil {
		MessageLogs.ErrorLog.Printf("an error while setting user because of %v\n", err)
		return err

	}

	// Add username to set SAdd() so that it will not be duplicated in future
	_, err = c.SAdd(ctx, usernameUniqueKey(id), user.Email).Result()
	if err != nil {
		MessageLogs.ErrorLog.Printf("Could not ")
		return err
	}

	// By using sorted set ZAdd(), every user will have a score associated with time which will sort the users
	var t = float64(time.Now().Unix())

	_, err = c.ZAdd(ctx, usernameUniqueKey(id), redis.Z{Member: user.Email, Score: t}).Result()
	if err != nil {
		MessageLogs.ErrorLog.Printf("an error of occured because %v\n", err)
		return err

	}

	return nil
}
