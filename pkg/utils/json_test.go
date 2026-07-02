package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bicosteve/booking-system/entities"
	"github.com/stretchr/testify/assert"
)

type sample struct {
	Name string `json:"name"`
}

func TestSerializeJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid single object",
			body:    `{"name":"john"}`,
			wantErr: false,
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
			errMsg:  "body must not be empty",
		},
		{
			name:    "badly formed json",
			body:    `{"name":`,
			wantErr: true,
		},
		{
			name:    "unknown field",
			body:    `{"name":"john","age":30}`,
			wantErr: true,
		},
		{
			name:    "multiple json objects",
			body:    `{"name":"john"}{"name":"jane"}`,
			wantErr: true,
			errMsg:  "body must only be a single json",
		},
		{
			name:    "wrong type",
			body:    `{"name":123}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			var data sample
			err := SerializeJSON(w, r, &data)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.EqualError(t, err, tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "john", data.Name)
			}
		})
	}
}

func TestDeserializeJSON(t *testing.T) {
	t.Run("writes status and body", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := sample{Name: "jane"}

		err := DeserializeJSON(w, http.StatusCreated, payload)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var got sample
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, "jane", got.Name)
	})

	t.Run("applies custom headers", func(t *testing.T) {
		w := httptest.NewRecorder()
		headers := http.Header{}
		headers.Set("X-Custom", "value")

		err := DeserializeJSON(w, http.StatusOK, sample{Name: "a"}, headers)
		assert.NoError(t, err)
		assert.Equal(t, "value", w.Header().Get("X-Custom"))
	})

	t.Run("unmarshalable payload returns error", func(t *testing.T) {
		w := httptest.NewRecorder()
		// channels cannot be marshaled to JSON
		err := DeserializeJSON(w, http.StatusOK, make(chan int))
		assert.Error(t, err)
	})
}

func TestErrorJSON(t *testing.T) {
	t.Run("default status is 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := ErrorJSON(w, errors.New("boom"))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp entities.JSONResponse
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.True(t, resp.Error)
		assert.Equal(t, "boom", resp.Message)
	})

	t.Run("custom status", func(t *testing.T) {
		w := httptest.NewRecorder()
		err := ErrorJSON(w, errors.New("nope"), http.StatusUnauthorized)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSerializeJSON_TooLarge(t *testing.T) {
	// build a body larger than the 1MB max
	big := bytes.Repeat([]byte("a"), 1_048_577)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(append([]byte(`{"name":"`), append(big, []byte(`"}`)...)...)))
	w := httptest.NewRecorder()

	var data sample
	err := SerializeJSON(w, r, &data)
	assert.Error(t, err)
}
