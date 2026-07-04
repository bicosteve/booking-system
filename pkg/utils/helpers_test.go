package utils

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

func intPtr(i int) *int { return &i }
func f64Ptr(f float64) *float64 {
	return &f
}

func TestValidateUser(t *testing.T) {
	tests := []struct {
		name    string
		payload entities.UserPayload
		wantErr string
	}{
		{
			name: "valid user",
			payload: entities.UserPayload{
				Email:           "user@example.com",
				PhoneNumber:     "0700000000",
				IsVendor:        "NO",
				Password:        "secret",
				ConfirmPassword: "secret",
			},
			wantErr: "",
		},
		{
			name:    "missing email",
			payload: entities.UserPayload{},
			wantErr: "email is required",
		},
		{
			name: "invalid email",
			payload: entities.UserPayload{
				Email: "not-an-email",
			},
			wantErr: "valid email needed",
		},
		{
			name: "missing phone",
			payload: entities.UserPayload{
				Email: "user@example.com",
			},
			wantErr: "phone number is required",
		},
		{
			name: "missing isVendor",
			payload: entities.UserPayload{
				Email:       "user@example.com",
				PhoneNumber: "0700000000",
			},
			wantErr: "isVendor is required",
		},
		{
			name: "missing password",
			payload: entities.UserPayload{
				Email:       "user@example.com",
				PhoneNumber: "0700000000",
				IsVendor:    "NO",
			},
			wantErr: "password is required",
		},
		{
			name: "missing confirm password",
			payload: entities.UserPayload{
				Email:       "user@example.com",
				PhoneNumber: "0700000000",
				IsVendor:    "NO",
				Password:    "secret",
			},
			wantErr: "confirm password is required",
		},
		{
			name: "passwords do not match",
			payload: entities.UserPayload{
				Email:           "user@example.com",
				PhoneNumber:     "0700000000",
				IsVendor:        "NO",
				Password:        "secret",
				ConfirmPassword: "different",
			},
			wantErr: "password and confirm password is must match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.payload
			err := ValidateUser(&p)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name    string
		payload entities.UserPayload
		wantErr string
	}{
		{
			name:    "valid login",
			payload: entities.UserPayload{Email: "user@example.com", Password: "secret"},
			wantErr: "",
		},
		{
			name:    "missing email",
			payload: entities.UserPayload{},
			wantErr: "email is required",
		},
		{
			name:    "invalid email",
			payload: entities.UserPayload{Email: "bad"},
			wantErr: "valid email needed",
		},
		{
			name:    "missing password",
			payload: entities.UserPayload{Email: "user@example.com"},
			wantErr: "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.payload
			err := ValidateLogin(&p)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRoom(t *testing.T) {
	tests := []struct {
		name    string
		payload entities.RoomPayload
		wantErr string
	}{
		{
			name:    "valid room",
			payload: entities.RoomPayload{Cost: "100", Status: "VACANT"},
			wantErr: "",
		},
		{
			name:    "missing cost",
			payload: entities.RoomPayload{Status: "VACANT"},
			wantErr: "room cost is required",
		},
		{
			name:    "missing status",
			payload: entities.RoomPayload{Cost: "100"},
			wantErr: "room status required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.payload
			err := ValidateRoom(&p)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBooking(t *testing.T) {
	tests := []struct {
		name    string
		payload entities.BookingPayload
		wantErr string
	}{
		{
			name: "valid booking",
			payload: entities.BookingPayload{
				Days:   intPtr(2),
				RoomID: intPtr(1),
				Amount: f64Ptr(100),
			},
			wantErr: "",
		},
		{
			name:    "missing days",
			payload: entities.BookingPayload{},
			wantErr: "days is required",
		},
		{
			name:    "missing room id",
			payload: entities.BookingPayload{Days: intPtr(2)},
			wantErr: "room id is required",
		},
		{
			name:    "missing amount",
			payload: entities.BookingPayload{Days: intPtr(2), RoomID: intPtr(1)},
			wantErr: "amount is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.payload
			err := ValidateBooking(&p)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestGeneratePasswordHashAndCompare(t *testing.T) {
	hash, err := GeneratePasswordHash("mypassword")
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "mypassword", hash)

	// correct password matches
	assert.True(t, ComparePasswordWithHash("mypassword", &hash))

	// wrong password does not match
	assert.False(t, ComparePasswordWithHash("wrongpassword", &hash))
}

func TestComparePasswordWithHash_InvalidHash(t *testing.T) {
	invalid := "not-a-bcrypt-hash"
	assert.False(t, ComparePasswordWithHash("anything", &invalid))
}

func TestGenerateAuthToken(t *testing.T) {
	user := entities.User{
		ID:          "1",
		Email:       "user@example.com",
		IsVender:    "NO",
		PhoneNumber: "0700000000",
	}

	token, err := GenerateAuthToken(user, "secret")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// A JWT has three dot-separated segments.
	parts := strings.Split(token, ".")
	assert.Len(t, parts, 3)
}

func TestVerifyAuthToken(t *testing.T) {
	user := entities.User{
		ID:          "7",
		Email:       "user@example.com",
		IsVender:    "YES",
		PhoneNumber: "0700000000",
	}
	secret := "topsecret"

	token, err := GenerateAuthToken(user, secret)
	assert.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := verifyAuthToken(token, secret)
		assert.NoError(t, err)
		assert.Equal(t, "user@example.com", claims.Username)
		assert.Equal(t, "7", claims.UserID)
		assert.Equal(t, "YES", claims.IsVendor)
	})

	t.Run("wrong secret", func(t *testing.T) {
		_, err := verifyAuthToken(token, "wrong-secret")
		assert.Error(t, err)
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := verifyAuthToken("garbage.token.value", secret)
		assert.Error(t, err)
	})
}

func TestGenerateResetToken(t *testing.T) {
	token, err := GenerateResetToken("42")
	assert.NoError(t, err)

	parts := strings.Split(token, "|")
	assert.Len(t, parts, 3)
	assert.Equal(t, "42", parts[2])
}

func TestIsValidResetToken(t *testing.T) {
	t.Run("valid unexpired token", func(t *testing.T) {
		token, err := GenerateResetToken("99")
		assert.NoError(t, err)

		valid, userID, err := IsValidResetToken(token)
		assert.NoError(t, err)
		assert.True(t, valid)
		assert.Equal(t, "99", userID)
	})

	t.Run("expired token", func(t *testing.T) {
		past := time.Now().UTC().Add(-1 * time.Hour).UnixMilli()
		token := "sometoken|" + strconv.FormatInt(past, 10) + "|55"

		valid, userID, err := IsValidResetToken(token)
		assert.NoError(t, err)
		assert.False(t, valid)
		assert.Empty(t, userID)
	})

	t.Run("malformed token - too few parts", func(t *testing.T) {
		valid, _, err := IsValidResetToken("onlyonepart")
		assert.Error(t, err)
		assert.False(t, valid)
	})

	t.Run("non numeric expiry", func(t *testing.T) {
		valid, _, err := IsValidResetToken("tok|notanumber|1")
		assert.Error(t, err)
		assert.False(t, valid)
	})
}

func TestValidateFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters entities.Filters
		wantErr string
	}{
		{
			name:    "valid with allowed sort",
			filters: entities.Filters{Page: 1, PageSize: 10, Sort: "cost"},
			wantErr: "",
		},
		{
			name:    "valid with empty sort",
			filters: entities.Filters{Page: 1, PageSize: 10, Sort: ""},
			wantErr: "",
		},
		{
			name:    "valid with id sort",
			filters: entities.Filters{Page: 1, PageSize: 10, Sort: "id"},
			wantErr: "",
		},
		{
			name:    "valid with created_at sort",
			filters: entities.Filters{Page: 1, PageSize: 10, Sort: "created_at"},
			wantErr: "",
		},
		{
			name:    "page too large",
			filters: entities.Filters{Page: 101, PageSize: 10, Sort: "id"},
			wantErr: "page must be between 1 and 100",
		},
		{
			name:    "page negative",
			filters: entities.Filters{Page: -1},
			wantErr: "page must be between 1 and 100",
		},
		{
			name:    "page size too large",
			filters: entities.Filters{Page: 1, PageSize: 21, Sort: "id"},
			wantErr: "page size must be between 1 and 20",
		},
		{
			name:    "disallowed sort",
			filters: entities.Filters{Page: 1, PageSize: 10, Sort: "name"},
			wantErr: "provided sort parameter is not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilters(tt.filters)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestFilterRoomByID(t *testing.T) {
	rooms := []*entities.Room{
		{ID: "1", Status: "VACANT"},
		{ID: "2", Status: "BOOKED"},
		{ID: "3", Status: "VACANT"},
	}

	t.Run("found", func(t *testing.T) {
		room, found := FilterRoomByID(rooms, "2")
		assert.True(t, found)
		assert.Equal(t, "2", room.ID)
	})

	t.Run("not found", func(t *testing.T) {
		room, found := FilterRoomByID(rooms, "99")
		assert.False(t, found)
		assert.Equal(t, &entities.Room{}, room)
	})
}

func TestFilterRoomByStatus(t *testing.T) {
	rooms := []*entities.Room{
		{ID: "1", Status: "VACANT"},
		{ID: "2", Status: "BOOKED"},
		{ID: "3", Status: "VACANT"},
	}

	t.Run("returns all matching vacant rooms", func(t *testing.T) {
		result, found := FilterRoomByStatus(rooms, "VACANT")
		assert.True(t, found)
		assert.Len(t, result, 2)
	})

	t.Run("returns all matching booked rooms", func(t *testing.T) {
		result, found := FilterRoomByStatus(rooms, "BOOKED")
		assert.True(t, found)
		assert.Len(t, result, 1)
	})

	t.Run("invalid status", func(t *testing.T) {
		result, found := FilterRoomByStatus(rooms, "UNKNOWN")
		assert.False(t, found)
		assert.Nil(t, result)
	})

	t.Run("no matches", func(t *testing.T) {
		onlyBooked := []*entities.Room{{ID: "9", Status: "BOOKED"}}
		result, found := FilterRoomByStatus(onlyBooked, "VACANT")
		assert.False(t, found)
		assert.Nil(t, result)
	})
}
