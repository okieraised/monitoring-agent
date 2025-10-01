package log

import (
	"testing"

	"go.uber.org/zap"
)

func TestInitDefault(t *testing.T) {
	Default().Error("test init default")
}

func TestLogger_With(t *testing.T) {
	l := Default()

	l1 := l.With(zap.String("x", "y"))
	l1.Info("test logger 1")

	l2 := l.With(zap.String("x2", "y2"))
	l2.Info("test logger 2")

	l1.Info("test logger 111")
}
