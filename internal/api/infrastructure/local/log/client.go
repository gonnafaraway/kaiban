package log

import "go.uber.org/zap"

func NewLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

func NewFallbackLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}
