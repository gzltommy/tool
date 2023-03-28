package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"tool-attendance/config"

	"github.com/sirupsen/logrus"
)

type customLog struct {
	*logrus.Logger
}

var Log *customLog
var serverName string

func defaultConfig() *config.LoggerConfig {
	return &config.LoggerConfig{
		Level:        "info",
		Write:        true,
		Path:         "./runtime/logs/",
		FileName:     "daily.log",
		MaxAge:       24,
		RotationTime: 7 * 24,
		Debug:        false,
	}
}

func Init(config *config.LoggerConfig) error {
	hostName, err := os.Hostname()
	if err == nil {
		tempArr := strings.Split(hostName, "-")
		if len(tempArr) > 2 {
			serverName = strings.Join(tempArr[:len(tempArr)-2], "-")
		} else {
			serverName = hostName
		}
	}
	if Log != nil {
		return nil
	}
	defaultCfg := defaultConfig()
	if config == nil {
		config = defaultCfg
	}

	Log = &customLog{logrus.New()}
	level := config.Level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}

	logDir := config.Path
	if logDir == "" {
		logDir = defaultCfg.Path
	}

	logFileName := config.FileName
	if logFileName == "" {
		logFileName = defaultCfg.FileName
	}

	Log.SetLevel(logLevel)
	Log.SetReportCaller(true)

	formatter := &customerFormatter{}
	if config.Write {
		storeLogDir := logDir
		err = os.MkdirAll(storeLogDir, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("creating log file failed: %s", err.Error()))
		}

		filePath := filepath.Join(storeLogDir, logFileName)
		writer, err := rotatelogs.New(
			filePath+".%Y-%m-%d",
			rotatelogs.WithClock(rotatelogs.Local),
			rotatelogs.WithMaxAge(config.MaxAge*time.Hour),
			rotatelogs.WithRotationTime(config.RotationTime*time.Hour),
			rotatelogs.WithLocation(time.FixedZone("CST", 8*3600)),
			rotatelogs.WithLinkName(filePath),
		)
		if err != nil {
			panic(fmt.Sprintf("rotatelogs log failed: %s", err.Error()))
		}
		Log.Out = writer
		Log.Formatter = formatter
	} else {
		Log.Out = os.Stdout
		Log.Formatter = formatter
	}
	return nil
}

func (c *customLog) WithAlarm() *logrus.Entry {
	return c.WithField("alarm", true)
}

type customerFormatter struct{}

func (c *customerFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	newLog := make(map[string]interface{}, len(entry.Data)+5)

	var cstZone = time.FixedZone("CST", 8*3600)
	newLog["time"] = entry.Time.In(cstZone).Format("2006-01-02 15:04:05")
	newLog["level"] = entry.Level.String()
	if entry.HasCaller() {
		fileName := path.Base(entry.Caller.File)
		newLog["func"] = fmt.Sprintf("%s(%s:%d)", path.Base(entry.Caller.Function), fileName, entry.Caller.Line)
	}
	newLog["msg"] = entry.Message
	newLog["alarm"] = false
	newLog["server_name"] = serverName

	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			newLog[k] = v.Error()
		default:
			newLog[k] = v
		}
	}

	logBytes, err := json.Marshal(newLog)
	if err != nil {
		return b.Bytes(), err
	}
	b.Write(logBytes)
	b.WriteByte('\r')
	b.WriteByte('\n')
	return b.Bytes(), nil
}
