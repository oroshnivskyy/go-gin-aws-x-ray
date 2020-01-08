package xray

import (
	"bytes"
	"context"
	"net/http"
	"strconv"

	"github.com/aws/aws-xray-sdk-go/xray"
)

func TraceOutgoingRequest(ctx context.Context, req *http.Request) {
	seg := xray.GetSegment(ctx)

	var respHeader bytes.Buffer
	{
		respHeader.WriteString("Root=")
		respHeader.WriteString(traceID(seg))
		respHeader.WriteString(";Parent=")
		respHeader.WriteString(seg.ID)
		respHeader.WriteString(";Sampled=")
		respHeader.WriteString(strconv.Itoa(btoi(seg.Sampled)))
	}
	req.Header.Set("x-amzn-trace-id", respHeader.String())
}

func traceID(seg *xray.Segment) string {
	if seg.TraceID == "" && seg.ParentSegment != nil {
		return traceID(seg.ParentSegment)
	}

	return seg.TraceID
}
