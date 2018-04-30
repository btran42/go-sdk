package logger

import (
	"log"
	"os"
)

// RedirectLog redirects the system logger.
func RedirectLog(l *Logger) (reset func()) {
	flags := log.Flags()
	prefix := log.Prefix()
	log.SetFlags(0)
	log.SetPrefix("")
	log.SetOutput(&logRedirect{l})
	return func() {
		log.SetFlags(flags)
		log.SetPrefix(prefix)
		log.SetOutput(os.Stdout)
	}
}

type logRedirect struct {
	*Logger
}

func (lr *logRedirect) Write(contents []byte) (int, error) {
	lr.SyncInfof(string(contents))
	return len(contents), nil
}
