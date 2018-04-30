package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/logger"
	"go.uber.org/zap"
)

var pool = logger.NewBufferPool(1024)

func logged(log *logger.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rw := logger.NewResponseWriter(res)
		handler(rw, req)

		log.Trigger(logger.NewWebRequestEvent(req).WithStatusCode(rw.StatusCode()).WithContentLength(int64(rw.ContentLength())).WithElapsed(time.Now().Sub(start)))
	}
}

func zapLogged(zl *zap.Logger, handler http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rw := logger.NewResponseWriter(res)
		handler(rw, req)

		zl.Info("web requst",
			zap.String("remote addr", logger.GetIP(req)),
			zap.String("path", req.URL.Path),
			zap.Int("status code", rw.StatusCode()),
			zap.Int("content length", rw.ContentLength()),
			zap.String("elapsed", fmt.Sprintf("%v", time.Since(start))),
		)
	}
}

func stdoutLogged(handler http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rw := logger.NewResponseWriter(res)
		handler(rw, req)

		buf := pool.Get()
		buf.WriteString(time.Now().UTC().Format(time.RFC3339))
		buf.WriteRune(logger.RuneSpace)
		buf.WriteString("web.request")
		buf.WriteRune(logger.RuneSpace)
		buf.WriteString(req.Method)
		buf.WriteRune(logger.RuneSpace)
		buf.WriteString(req.URL.Path)
		buf.WriteRune(logger.RuneSpace)
		buf.WriteString(strconv.Itoa(rw.StatusCode()))
		buf.WriteRune(logger.RuneSpace)
		buf.WriteString(time.Since(start).String())
		buf.WriteRune(logger.RuneSpace)
		buf.WriteString(logger.FormatFileSize(int64(rw.ContentLength())))
		buf.WriteRune(logger.RuneNewline)
		os.Stdout.Write(buf.Bytes())
		pool.Put(buf)
	}
}

func indexHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`{"status":"ok!"}`))
}

func fatalErrorHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusInternalServerError)
	logger.Default().Fatal(exception.New("this is a fatal exception"))
	res.Write([]byte(`{"status":"not ok."}`))
}

func errorHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusInternalServerError)
	logger.Default().Error(exception.New("this is an exception"))
	res.Write([]byte(`{"status":"not ok."}`))
}

func warningHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusBadRequest)
	logger.Default().Warning(exception.New("this is warning"))
	res.Write([]byte(`{"status":"not ok."}`))
}

func subContextHandler(res http.ResponseWriter, req *http.Request) {
	logger.Default().SubContext("sub-context").SubContext("another one").Infof("called an endpoint we care about")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`{"status":"did sub-context things"}`))
}

func auditHandler(res http.ResponseWriter, req *http.Request) {
	logger.Default().Trigger(logger.NewAuditEvent(logger.GetIP(req), "viewed", "audit route").WithExtra(map[string]string{
		"remoteAddr": req.RemoteAddr,
		"host":       req.Host,
	}))
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`{"status":"audit logged!"}`))
}

func port() string {
	envPort := os.Getenv("PORT")
	if len(envPort) > 0 {
		return envPort
	}
	return "8888"
}

func main() {
	log := logger.New().
		WithFlags(logger.AllFlags()).
		WithWriter(logger.NewTextWriter(os.Stderr).WithUseColor(false))

	zl, _ := zap.NewProduction()

	http.HandleFunc("/", logged(log, indexHandler))

	http.HandleFunc("/sub-context", logged(log, subContextHandler))
	http.HandleFunc("/fatalerror", logged(log, fatalErrorHandler))
	http.HandleFunc("/error", logged(log, errorHandler))
	http.HandleFunc("/warning", logged(log, warningHandler))
	http.HandleFunc("/audit", logged(log, auditHandler))

	http.HandleFunc("/bench/logged", logged(log, indexHandler))
	http.HandleFunc("/bench/zap", zapLogged(zl, indexHandler))
	http.HandleFunc("/bench/stdout", stdoutLogged(indexHandler))

	logger.Default().Infof("Listening on :%s", port())
	logger.Default().Infof("Events %s", log.Flags().String())

	log.SyncFatalExit(http.ListenAndServe(":"+port(), nil))
}
