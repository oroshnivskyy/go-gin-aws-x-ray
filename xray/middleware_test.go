package xray

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testData struct {
	Code          int
	Body          string
	Method        string
	Path          string
	ClientIP      string
	ContentLength int
	headers       []test_header
	Error         bool
	Throttle      bool
	Fault         bool
}

func TestMiddlewareGeneralCase(t *testing.T) {

	tDaemon, e := NewXRayTestDaemon()
	if !assert.NoError(t, e) {
		return
	}
	go tDaemon.Run()
	defer tDaemon.Stop()
	router := gin.New()
	router.Use(Middleware())

	router.GET("/", func(c *gin.Context) {
		c.String(200, "OK.")
	})

	router.GET("/400", func(c *gin.Context) {
		c.String(400, "Bad Request")
	})

	router.POST("/429", func(c *gin.Context) {
		c.String(429, "Too many requests")
	})

	router.GET("/500", func(c *gin.Context) {
		c.String(500, "StatusInternalServerError")
	})

	tdata := []testData{
		{http.StatusOK, "OK.", "GET", "/", "192.0.2.1:1234", 3, []test_header{}, false, false, false},
		{http.StatusOK, "OK.", "GET", "/", "198.0.2.1", 3, []test_header{{"X-Forwarded-For", "198.0.2.1,127.0.0.1"}}, false, false, false},
		{http.StatusBadRequest, "Bad Request", "GET", "/400", "192.0.2.1:1234", 11, []test_header{}, true, false, false},
		{http.StatusTooManyRequests, "Too many requests", "POST", "/429", "192.0.2.1:1234", 17, []test_header{}, true, true, false},
		{http.StatusInternalServerError, "StatusInternalServerError", "GET", "/500", "192.0.2.1:1234", 25, []test_header{}, false, false, true},
	}
	for _, td := range tdata {
		w := performRequest(router, td.Method, td.Path, td.headers...)

		s, e := tDaemon.Recv()
		if assert.NoError(t, e) {
			// TEST
			assert.Equal(t, td.Code, w.Code, "Performing request %s", td.Path)
			assert.Equal(t, td.Body, w.Body.String(), "Performing request %s", td.Path)
			assert.NotEmpty(t, w.Header().Get(headerTraceID), "Performing request %s", td.Path)

			assert.Equal(t, td.Code, s.HTTP.Response.Status, "Performing request %s", td.Path)
			assert.Equal(t, td.Method, s.HTTP.Request.Method, "Performing request %s", td.Path)
			assert.Equal(t, td.Path, s.HTTP.Request.URL, "Performing request %s", td.Path)
			assert.Equal(t, td.ClientIP, s.HTTP.Request.ClientIP, "Performing request %s", td.Path)
			assert.Equal(t, "UnitTest", s.HTTP.Request.UserAgent, "Performing request %s", td.Path)
			assert.Equal(t, td.ContentLength, s.HTTP.Response.ContentLength, "Performing request %s", td.Path)
			assert.Equal(t, td.Error, s.Error, "Performing request %s", td.Path)
			assert.Equal(t, td.Throttle, s.Throttle, "Performing request %s", td.Path)
			assert.Equal(t, td.Fault, s.Fault, "Performing request %s", td.Path)
		}
	}

	tDaemon.Done = true
}

func performRequest(r http.Handler, method, path string, headers ...test_header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("User-Agent", "UnitTest")
	for _, h := range headers {
		req.Header.Add(h.Key, h.Value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type test_header struct {
	Key   string
	Value string
}
