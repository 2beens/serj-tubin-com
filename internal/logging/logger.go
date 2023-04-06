package logging

import (
	"log"
	"os"
	"strings"

	"github.com/2beens/serjtubincom/pkg"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerSetupParams struct {
	LogFileName   string
	LogToStdout   bool
	LogLevel      string
	LogFormatJSON bool
	Environment   string
	SentryEnabled bool
	SentryDSN     string
}

func Setup(params LoggerSetupParams) {
	logger := logrus.New()

	if params.LogFormatJSON {
		logger.Formatter = &logrus.JSONFormatter{}
	}

	if params.SentryEnabled {
		err := sentry.Init(sentry.ClientOptions{
			Environment: params.Environment,
			Dsn:         params.SentryDSN,
			// TODO: check if needed
			TracesSampleRate: 1.0,
		})
		if err != nil {
			logger.Errorf("sentry.Init: %s", err)
		}

		hook := NewSentryHook([]logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		})
		logger.AddHook(hook)

		sentry.CaptureMessage("Sentry set up successfully")
		logger.Infoln("Sentry set up successfully")
	}

	logger.SetLevel(GetLevel(params.LogLevel))

	if params.LogFileName == "" {
		logger.SetOutput(os.Stdout)
		logger.Println("writing logs only to STDOUT")
		return
	}

	if params.LogToStdout {
		logger.Println("writing logs to file and STDOUT")
	}

	if !strings.HasSuffix(params.LogFileName, ".log") {
		params.LogFileName += ".log"
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:  params.LogFileName,
		MaxSize:   50,    // megabytes
		LocalTime: false, // false -> use UTC
		Compress:  true,  // disabled by default
		// comment out MaxBackups and MaxAge, as I want to retain rotated log files indefinitely for now
		//MaxBackups: 30,
		//MaxAge:     730,   //days
	}

	if params.LogToStdout {
		logger.SetOutput(
			pkg.NewCombinedWriter(os.Stdout, lumberJackLogger),
		)
	} else {
		logger.SetOutput(lumberJackLogger)
	}

	// Use logrus for standard log output
	// Note that `log` here references stdlib's log
	// Not logrus imported under the name `log`.
	log.SetOutput(logger.Writer())
}

func GetLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "debug":
		return logrus.DebugLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "info":
		return logrus.InfoLevel
	case "trace":
		return logrus.TraceLevel
	case "warn":
		return logrus.WarnLevel
	default:
		return logrus.TraceLevel
	}
}
