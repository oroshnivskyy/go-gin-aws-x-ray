package xray

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/gin-gonic/gin"
)

const headerTraceID = "x-amzn-trace-id"

func Middleware(sn xray.SegmentNamer) gin.HandlerFunc {
	return func(c *gin.Context) {
		traceHeader := header.FromString(c.Request.Header.Get(headerTraceID))
		var name string
		if sn != nil {
			name = sn.Name(c.Request.Host)
		} else {
			name = methodPathSegmentName(c.Request)
		}
		ctx, seg := xray.NewSegmentFromHeader(c.Request.Context(), name, traceHeader)
		c.Request = c.Request.WithContext(ctx)

		captureRequestData(c, seg)
		c.Next()
		captureResponseData(c, seg)

		seg.Close(nil)
	}
}

// Write request data to segment
func captureRequestData(c *gin.Context, seg *xray.Segment) {
	r := c.Request
	seg.Lock()
	defer seg.Unlock()
	segmentRequest := seg.GetHTTP().GetRequest()
	segmentRequest.Method = r.Method
	segmentRequest.URL = r.URL.String()
	segmentRequest.XForwardedFor = hasXForwardedFor(r)
	segmentRequest.ClientIP = clientIP(r)
	segmentRequest.UserAgent = r.UserAgent()
	c.Writer.Header().Set(headerTraceID, createTraceHeader(r, seg))
}

// Write response data to segment
func captureResponseData(c *gin.Context, seg *xray.Segment) {
	respStatus := c.Writer.Status()

	seg.Lock()
	defer seg.Unlock()
	seg.GetHTTP().GetResponse().Status = respStatus
	seg.GetHTTP().GetResponse().ContentLength = c.Writer.Size()

	if respStatus >= 400 && respStatus < 500 {
		seg.Error = true
	}
	if respStatus == 429 {
		seg.Throttle = true
	}
	if respStatus >= 500 && respStatus < 600 {
		seg.Fault = true
	}
}

// Define route name by method and path
func methodPathSegmentName(r *http.Request) string {
	return fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
}

//Add tracing data to header
func createTraceHeader(r *http.Request, seg *xray.Segment) string {
	trace := parseHeaders(r.Header)
	if trace["Root"] != "" {
		seg.TraceID = trace["Root"]
		seg.RequestWasTraced = true
	}
	if trace["Parent"] != "" {
		seg.ParentID = trace["Parent"]
	}
	// Don't use the segment's header here as we only want to
	// send back the root and possibly sampled values.
	var respHeader bytes.Buffer
	respHeader.WriteString("Root=")
	respHeader.WriteString(seg.TraceID)

	seg.Sampled = trace["Sampled"] != "0"
	if trace["Sampled"] == "?" {
		respHeader.WriteString(";Sampled=")
		respHeader.WriteString(strconv.Itoa(btoi(seg.Sampled)))
	}
	return respHeader.String()
}

func hasXForwardedFor(r *http.Request) bool {
	return r.Header.Get("X-Forwarded-For") != ""
}

func clientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		return strings.TrimSpace(strings.Split(forwardedFor, ",")[0])
	}

	return r.RemoteAddr
}

func parseHeaders(h http.Header) map[string]string {
	m := map[string]string{}
	s := h.Get(headerTraceID)
	for _, c := range strings.Split(s, ";") {
		p := strings.SplitN(c, "=", 2)
		k := strings.TrimSpace(p[0])
		v := ""
		if len(p) > 1 {
			v = strings.TrimSpace(p[1])
		}
		m[k] = v
	}
	return m
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
