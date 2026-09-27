package log

import "go.uber.org/zap"

func NewLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

func NewFallbackLogger() *zap.Logger {
	l, err := zap.NewDevelopment()
	if err != nil {
		return zap.NewNop()
	}
	return l
}
