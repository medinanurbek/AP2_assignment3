package grpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// LoggingInterceptor logs every incoming request's method name and execution duration
func LoggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	log.Printf("gRPC Call: %s \n", info.FullMethod)

	resp, err := handler(ctx, req)

	duration := time.Since(start)
	log.Printf("gRPC Call Completed: %s, duration: %v, err: %v \n", info.FullMethod, duration, err)

	return resp, err
}
