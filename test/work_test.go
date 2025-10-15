package test

import (
	"myapp/handlers"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestModelInit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	panic("not implemented yet")
}

func TestWorkStylize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/user/work/ai/transfer", handlers.TransferByAIAgent)

	reader := strings.NewReader(`{"image_url": "https://example.com/image.jpg"}`)

	req, err := http.NewRequest("POST", "/user/work/ai/transfer", reader)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
}
