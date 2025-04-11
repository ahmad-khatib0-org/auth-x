package assertx

import (
	"testing"
	"time"
)

func TestEqualAsJSONExcept(t *testing.T) {
	a := map[string]any{"foo": "bar", "baz": "bar", "bar": "baz"}
	b := map[string]any{"foo": "bar", "baz": "bar", "bar": "not-baz"}

	EqualAsJSONExcept(t, a, b, []string{"bar"})
}

func TestTimeDifferenceLess(t *testing.T) {
	TimeDifferenceLess(t, time.Now(), time.Now().Add(time.Second), 2)
}
