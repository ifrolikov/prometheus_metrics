package grpcinterceptor

import (
	"context"
	"github.com/iancoleman/strcase"
	"github.com/ifrolikov/prometheus_metrics/v4/interfaces"
	"google.golang.org/grpc"
	"path"
	"regexp"
	"time"
)

var re = regexp.MustCompile(`[^\w\d]+`)

func NewMetricsTimerUnaryInterceptor(collector interfaces.Collector) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()

		resp, err := handler(ctx, req)

		method := path.Base(info.FullMethod)
		metricName := strcase.ToSnake(re.ReplaceAllString(method, "_"))

		hasError := "false"
		if err != nil {
			hasError = "true"
		}
		_ = collector.ObserveTimer(metricName, startTime, map[string]string{"has_error": hasError})
		return resp, err
	}
}

func NewMetricsTimerStreamInterceptor(collector interfaces.Collector) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		startTime := time.Now()

		err := handler(srv, stream)

		method := path.Base(info.FullMethod)
		metricName := strcase.ToSnake(re.ReplaceAllString(method, "_"))

		hasError := "false"
		if err != nil {
			hasError = "true"
		}
		_ = collector.ObserveTimer(metricName, startTime, map[string]string{"has_error": hasError})
		return err
	}
}
