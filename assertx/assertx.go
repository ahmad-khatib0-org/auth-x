package assertx

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"
)

func PrettifyJSONPayload(t testing.TB, payload any) string {
	t.Helper()

	o, err := json.MarshalIndent(payload, "", "  ")
	require.NoError(t, err)
	return string(o)
}

func EqualAsJSON(t testing.TB, expected, actual any, args ...any) {
	t.Helper()

	var eb, ab bytes.Buffer
	if len(args) == 0 {
		args = append(args, []any{PrettifyJSONPayload(t, actual)})
	}

	require.NoError(t, json.NewEncoder(&eb).Encode(expected), args...)
	require.NoError(t, json.NewEncoder(&ab).Encode(actual), args...)
	assert.JSONEq(t, strings.TrimSpace(eb.String()), strings.TrimSpace(ab.String()), args...)
}

func EqualAsJSONExcept(t testing.TB, expected, actual any, except []string, args ...any) {
	t.Helper()
	var eb, ab bytes.Buffer
	if len(args) == 0 {
		args = []any{PrettifyJSONPayload(t, actual)}
	}

	require.NoError(t, json.NewEncoder(&eb).Encode(expected), args...)
	require.NoError(t, json.NewEncoder(&ab).Encode(actual), args...)

	var err error
	ebs, abs := eb.String(), ab.String()
	for _, k := range except {
		ebs, err = sjson.Delete(ebs, k)
		require.NoError(t, err)

		abs, err = sjson.Delete(abs, k)
		require.NoError(t, err)
	}

	assert.JSONEq(t, strings.TrimSpace(ebs), strings.TrimSpace(abs), args...)
}

func TimeDifferenceLess(t testing.TB, t1, t2 time.Time, seconds int) {
	t.Helper()
	delta := math.Abs(float64(t1.Unix()) - float64(t2.Unix()))
	assert.Less(t, delta, float64(seconds))
}
