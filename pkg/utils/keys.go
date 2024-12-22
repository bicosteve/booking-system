package utils

import (
	"strconv"
	"time"
)

func UsernameUniqueKey(userId int64) string {
	return "username:unique#" + strconv.FormatInt(userId, 10)
}

func UsersKey(userId string) string {
	return "users#" + userId
}

func GenId() int64 {
	// Used in sorted set to give time value as int ordered
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
	milli := date.UnixMilli()

	return int64(milli)
}

// func serializeUser(user entities.UserPayload) *entities.SerializedUser {
// 	return &entities.SerializedUser{
// 		Email:    user.Email,
// 		Password: user.Password,
// 	}

// }

func ConvertIdToScore(id int64) float64 {
	unixSeconds := id / 1000
	unixNano := (id % 1000) * 1e6
	t := time.Unix(unixSeconds, unixNano)
	return float64(t.UnixNano()) / 1e9
}
