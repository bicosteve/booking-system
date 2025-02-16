package controllers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bicosteve/booking-system/entities"
	"github.com/bicosteve/booking-system/repo"
	"github.com/bicosteve/booking-system/service"
	"github.com/stretchr/testify/assert"
)

var base Base

func TestRegisterHandler(t *testing.T) {

	testUser := entities.User{}
	_ = testUser

	sql, _, err := sqlmock.New()
	assert.NoError(t, err)

	userRepository := repo.NewDBRepository(sql)
	userService := service.NewUserService(*userRepository)
	base.userService = userService

	var tests = []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Valid User",
			requestBody:    `{"email":"bicooosteve4@gmail.com","phone_number":"0704961750","is_vendor":"NO", "password":"1234","confirm_password":"1234"}`,
			expectedStatus: http.StatusCreated,
		},
	}

	// payload := strings.NewReader(`{
	// 	"email":"bicossteve4@gmail.com",
	// 	"phone_number":"0705961750",
	// 	"is_vendor":"NO",
	// 	"password":"1234",
	// 	"confirm_password":"1234"
	// }`)

	for _, test := range tests {
		// p, _ := json.Marshal(test.requestBody)
		var reader io.Reader = strings.NewReader(test.requestBody)
		req, err := http.NewRequest(http.MethodPost, "/register", reader)
		assert.NoError(t, err)

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(base.RegisterHandler)

		handler.ServeHTTP(rr, req)

		if test.expectedStatus != rr.Code {
			t.Errorf("%s: returned wrong status code; expected %d but got %d", test.name, test.expectedStatus, rr.Code)
		}

	}

}
