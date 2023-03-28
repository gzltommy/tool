package log

import (
	"os"
	"strings"
	"testing"
	"tool-attendance/config"
)

func TestFormatLog(t *testing.T) {
	Init(&config.LoggerConfig{
		Level:          "info",
		Formatter:      "json",
		DisableConsole: true,
		Write:          true,
		Path:           "F:/桌面",
		FileName:       "aaa",
		MaxAge:         24,
		RotationTime:   7 * 24,
		Debug:          false,
		ReportCaller:   true,
	})
	Log.WithAlarm().Error("asdfasdfsdfsdfs", 234234)
}

func TestGetServerName(t *testing.T) {
	hostName, err := os.Hostname()
	hostName = "admin_server"
	if err == nil {
		tempArr := strings.Split(hostName, "-")
		if len(tempArr) > 2 {
			serverName = strings.Join(tempArr[:len(tempArr)-2], "-")
		} else {
			serverName = hostName
		}
	}
	t.Log(serverName)
}
