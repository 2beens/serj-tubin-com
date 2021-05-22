package logging

import (
	"os"
	"strings"

	"github.com/2beens/serjtubincom/pkg"
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func Setup(logFileName string, logToStdout bool, logLevel string) {
	log.SetLevel(GetLevel(logLevel))

	if logFileName == "" {
		log.SetOutput(os.Stdout)
		log.Println("writing logs only to STDOUT")
		return
	}

	if logToStdout {
		log.Println("writing logs to file and STDOUT")
	}

	if !strings.HasSuffix(logFileName, ".log") {
		logFileName += ".log"
	}

	lumberJackLogger := &lumberjack.Logger{
		Filename:  logFileName,
		MaxSize:   50,    // megabytes
		LocalTime: false, // false -> use UTC
		Compress:  true,  // disabled by default
		// comment out MaxBackups and MaxAge, as I want to retain rotated log files indefinitely for now
		//MaxBackups: 30,
		//MaxAge:     730,   //days
	}

	if logToStdout {
		log.SetOutput(pkg.NewCombinedWriter(os.Stdout, lumberJackLogger))
	} else {
		log.SetOutput(lumberJackLogger)
	}
}

func GetLevel(level string) log.Level {
	switch strings.ToLower(level) {
	case "debug":
		return log.DebugLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	case "info":
		return log.InfoLevel
	case "trace":
		return log.TraceLevel
	case "warn":
		return log.WarnLevel
	default:
		return log.TraceLevel
	}
}
