package common

import (
	"context"

	"github.com/aws/aws-xray-sdk-go/xray"
)

func WithTracing(ctx context.Context, segmentName string, fn func(context.Context) error) error {
	tracedCtx, seg := xray.BeginSubsegment(ctx, segmentName)
	defer seg.Close(nil)

	err := fn(tracedCtx)
	if err != nil {
		seg.AddError(err)
	}

	return err
}

func WithAnnotation(ctx context.Context, key string, value interface{}) {
	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddAnnotation(key, value)
	}
}

func WithMetadata(ctx context.Context, key string, value interface{}) {
	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddMetadata(key, value)
	}
}

func AddError(ctx context.Context, err error) {
	if seg := xray.GetSegment(ctx); seg != nil {
		seg.AddError(err)
	}
}