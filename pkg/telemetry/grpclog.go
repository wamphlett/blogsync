package telemetry

import (
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/grpc/grpclog"
)

// SetGRPCLogger redirects gRPC's internal logger to slog so that errors like
// "failed to create fsnotify watcher" (raised by the OTLP gRPC exporters)
// carry a proper level in structured logs instead of going straight to
// stderr unstructured.
func SetGRPCLogger() {
	grpclog.SetLoggerV2(&grpcSlogLogger{})
}

type grpcSlogLogger struct{}

func (g *grpcSlogLogger) Info(args ...interface{})   { slog.Info(fmt.Sprint(args...)) }
func (g *grpcSlogLogger) Infoln(args ...interface{}) { slog.Info(fmt.Sprint(args...)) }
func (g *grpcSlogLogger) Infof(format string, args ...interface{}) {
	slog.Info(fmt.Sprintf(format, args...))
}
func (g *grpcSlogLogger) Warning(args ...interface{})   { slog.Warn(fmt.Sprint(args...)) }
func (g *grpcSlogLogger) Warningln(args ...interface{}) { slog.Warn(fmt.Sprint(args...)) }
func (g *grpcSlogLogger) Warningf(format string, args ...interface{}) {
	slog.Warn(fmt.Sprintf(format, args...))
}
func (g *grpcSlogLogger) Error(args ...interface{})   { slog.Error(fmt.Sprint(args...)) }
func (g *grpcSlogLogger) Errorln(args ...interface{}) { slog.Error(fmt.Sprint(args...)) }
func (g *grpcSlogLogger) Errorf(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
}
func (g *grpcSlogLogger) Fatal(args ...interface{}) {
	slog.Error(fmt.Sprint(args...))
	os.Exit(1)
}
func (g *grpcSlogLogger) Fatalln(args ...interface{}) {
	slog.Error(fmt.Sprint(args...))
	os.Exit(1)
}
func (g *grpcSlogLogger) Fatalf(format string, args ...interface{}) {
	slog.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}
func (g *grpcSlogLogger) V(int) bool { return false }
