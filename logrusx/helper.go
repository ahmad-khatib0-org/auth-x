package logrusx

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/ahmad-khatib0-org/auth-x/errorsx"
	"github.com/gobuffalo/pop/v6/logging"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/httptrace/otelhttptrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type Logger struct {
	*logrus.Entry
	leakSensitive bool
	redactionText string
	opts          []Option
	name          string
	version       string
}

var opts = otelhttptrace.WithPropagators(
	propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
)

func (l *Logger) LeakSensitiveData() bool {
	return l.leakSensitive
}

func (l *Logger) Logrus() *logrus.Logger {
	return l.Entry.Logger
}

func (l *Logger) NewEntry() *Logger {
	ll := *l
	ll.Entry = logrus.NewEntry(l.Logger)
	return &ll
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	ll := *l
	ll.Entry = l.Logger.WithContext(ctx)
	return &ll
}

func (l *Logger) HTTPHeadersRedacted(h http.Header) map[string]any {
	headers := map[string]any{}
	for k, v := range h {
		switch kl := strings.ToLower(k); kl {
		case "authorization", "cookie", "set-cookie", "x-session-token":
			headers[kl] = l.maybeRedact(v)
		case "location":
			locationUrl, err := url.Parse(h.Get("Location"))
			if err != nil {
				headers[kl] = l.maybeRedact(v)
				continue
			}
			if l.leakSensitive {
				headers[kl] = locationUrl.String()
			} else {
				locationUrl.RawQuery = ""
				locationUrl.Fragment = ""
				headers[kl] = locationUrl.Redacted()
			}
		default:
			headers[kl] = h.Get(k)
		}
	}

	return headers
}

func (l *Logger) maybeRedact(value any) any {
	if fmt.Sprintf("%v", value) == "" || value == nil {
		return nil
	}
	if !l.leakSensitive {
		return l.redactionText
	}
	return value
}

func (l *Logger) WithRequest(r *http.Request) *Logger {
	headers := l.HTTPHeadersRedacted(r.Header)
	if ua := r.UserAgent(); len(ua) > 0 {
		headers["user-agent"] = ua
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}

	ll := l.WithField("http_request", map[string]any{
		"remote":  r.RemoteAddr,
		"method":  r.Method,
		"path":    r.URL.EscapedPath(),
		"query":   l.maybeRedact(r.URL.RawQuery),
		"scheme":  scheme,
		"host":    r.Host,
		"headers": headers,
	})

	spanCtx := trace.SpanContextFromContext(r.Context())
	if !spanCtx.IsValid() {
		_, _, spanCtx = otelhttptrace.Extract(r.Context(), r, opts)
	}
	if spanCtx.IsValid() {
		traces := make(map[string]string, 2)
		if spanCtx.HasTraceID() {
			traces["trace_id"] = spanCtx.TraceID().String()
		}
		if spanCtx.HasSpanID() {
			traces["span_id"] = spanCtx.SpanID().String()
		}
		ll = ll.WithField("otel", traces)
	}

	return ll
}

func (l *Logger) WithSpanFromContext(ctx context.Context) *Logger {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return l
	}

	traces := make(map[string]string, 2)
	if spanCtx.HasTraceID() {
		traces["trace_id"] = spanCtx.TraceID().String()
	}
	if spanCtx.HasSpanID() {
		traces["span_id"] = spanCtx.SpanID().String()
	}

	return l.WithField("otel", traces)
}

func (l *Logger) Logf(level logrus.Level, format string, args ...any) {
	if !l.leakSensitive {
		for i, arg := range args {
			switch urlArg := arg.(type) {
			case url.URL:
				urlCopy := url.URL{Scheme: urlArg.Scheme, Host: urlArg.Host, Path: urlArg.Path}
				args[i] = urlCopy
			case *url.URL:
				urlCopy := url.URL{Scheme: urlArg.Scheme, Host: urlArg.Host, Path: urlArg.Path}
				args[i] = &urlCopy
			default:
				continue
			}
		}
	}
	l.Entry.Logf(level, format, args...)
}

func (l *Logger) Tracef(format string, args ...any) {
	l.Logf(logrus.TraceLevel, format, args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	l.Logf(logrus.DebugLevel, format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	l.Logf(logrus.InfoLevel, format, args...)
}

func (l *Logger) Printf(format string, args ...any) {
	l.Infof(format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.Logf(logrus.WarnLevel, format, args...)
}

func (l *Logger) Warningf(format string, args ...any) {
	l.Warnf(format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.Logf(logrus.ErrorLevel, format, args...)
}

func (l *Logger) Fatalf(format string, args ...any) {
	l.Logf(logrus.FatalLevel, format, args...)
	l.Entry.Logger.Exit(1)
}

func (l *Logger) Panicf(format string, args ...any) {
	l.Logf(logrus.PanicLevel, format, args...)
}

func (l *Logger) WithFields(f logrus.Fields) *Logger {
	ll := *l
	ll.Entry = l.Entry.WithFields(f)
	return &ll
}

func (l *Logger) WithField(key string, value any) *Logger {
	ll := *l
	ll.Entry = l.Entry.WithField(key, value)
	return &ll
}

func (l *Logger) WithSensitiveField(key string, value any) *Logger {
	return l.WithField(key, l.maybeRedact(value))
}

func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}

	ctx := map[string]any{"message": err.Error()}
	if l.Entry.Logger.IsLevelEnabled(logrus.DebugLevel) {
		if e, ok := err.(errorsx.StackTracer); ok {
			ctx["stack_trace"] = fmt.Sprintf("%+v", e.StackTrace())
		} else {
			ctx["stack_trace"] = fmt.Sprintf("stack trace could not be recovered from error type %s", reflect.TypeOf(err))
		}
	}
	if c := errorsx.ReasonCarrier(nil); errors.As(err, &c) {
		ctx["reason"] = c.Reason()
	}
	if c := errorsx.RequestIDCarrier(nil); errors.As(err, &c) && c.RequestID() != "" {
		ctx["request_id"] = c.RequestID()
	}
	if c := errorsx.DetailsCarrier(nil); errors.As(err, &c) && c.Details() != nil {
		ctx["details"] = c.Details()
	}
	if c := errorsx.StatusCarrier(nil); errors.As(err, &c) && c.Status() != "" {
		ctx["status"] = c.Status()
	}
	if c := errorsx.StatusCodeCarrier(nil); errors.As(err, &c) && c.StatusCode() != 0 {
		ctx["status_code"] = c.StatusCode()
	}
	if c := errorsx.DebugCarrier(nil); errors.As(err, &c) {
		ctx["debug"] = c.Debug()
	}

	return l.WithField("error", ctx)
}

var popLevelTranslations = map[logging.Level]logrus.Level{
	// logging.SQL:   logrus.TraceLevel, we never want to log SQL statements, see https://github.com/ory/keto/issues/454
	logging.Debug: logrus.DebugLevel,
	logging.Info:  logrus.InfoLevel,
	logging.Warn:  logrus.WarnLevel,
	logging.Error: logrus.ErrorLevel,
}

func (l *Logger) PopLogger(lvl logging.Level, s string, args ...any) {
	level, ok := popLevelTranslations[lvl]
	if ok {
		l.WithField("source", "pop").Logf(level, s, args...)
	}
}
