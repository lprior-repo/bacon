package common

import (
	"context"
	"os"

	"github.com/aws/aws-xray-sdk-go/v2/xray"
)

// TestContext creates a context with X-Ray tracing enabled for testing
// This initializes a root segment so that subsegments can be created
func TestContext(segmentName string) (context.Context, func()) {
	// Set X-Ray to use the test context plugin to avoid AWS Lambda requirements
	os.Setenv("_X_AMZN_TRACE_ID", "Root=1-5e1b4151-5ac6c58a52934f1124456789;Parent=1234567890123456;Sampled=1")
	
	ctx := context.Background()
	ctx, seg := xray.BeginSegment(ctx, segmentName)
	
	// Return cleanup function
	cleanup := func() {
		seg.Close(nil)
		os.Unsetenv("_X_AMZN_TRACE_ID")
	}
	
	return ctx, cleanup
}

// TestContextWithSubsegment creates a context with both root segment and subsegment for testing
func TestContextWithSubsegment(rootName, subsegmentName string) (context.Context, func()) {
	ctx, rootCleanup := TestContext(rootName)
	
	ctx, subseg := xray.BeginSubsegment(ctx, subsegmentName)
	
	// Return cleanup function that closes both
	cleanup := func() {
		subseg.Close(nil)
		rootCleanup()
	}
	
	return ctx, cleanup
}