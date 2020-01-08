package xray

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/stretchr/testify/assert"
)

func TestMasterSegment(t *testing.T) {
	ctx, seg := xray.BeginSegment(context.TODO(), "Test")
	req, _ := http.NewRequest("POST", "/", strings.NewReader("{}"))

	TraceOutgoingRequest(ctx, req)

	expectedSampled := 0
	if seg.Sampled {
		expectedSampled = 1
	}

	assert.Equal(
		t,
		fmt.Sprintf("Root=%s;Parent=%s;Sampled=%d", seg.TraceID, seg.ID, expectedSampled),
		req.Header.Get("x-amzn-trace-id"),
	)
}

func TestSubsegment(t *testing.T) {
	ctx, seg := xray.BeginSegment(context.TODO(), "Test")
	ctx1, seg1 := xray.BeginSubsegment(ctx, "TestSubsegment")
	req, _ := http.NewRequest("POST", "/", strings.NewReader("{}"))

	TraceOutgoingRequest(ctx1, req)

	expectedSampled := 0
	if seg1.Sampled {
		expectedSampled = 1
	}

	assert.Equal(
		t,
		fmt.Sprintf("Root=%s;Parent=%s;Sampled=%d", seg.TraceID, seg1.ID, expectedSampled),
		req.Header.Get("x-amzn-trace-id"),
	)
}
