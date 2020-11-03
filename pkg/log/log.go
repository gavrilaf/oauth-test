package log

import (
	"context"

	"github.com/sirupsen/logrus"
)

func newLoggerEntry() *logrus.Entry {
	logger := logrus.StandardLogger()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})

	return logrus.NewEntry(logger)
}

var (
	L = newLoggerEntry()
)

type logKey struct{}

func InitLog() {
	L = newLoggerEntry()
}

func FromContext(ctx context.Context) *logrus.Entry {
	entry := ctx.Value(logKey{})
	if entry == nil {
		return L
	}

	return entry.(*logrus.Entry)
}

func WithLogger(ctx context.Context, entry *logrus.Entry) context.Context {
	return context.WithValue(ctx, logKey{}, entry)
}
