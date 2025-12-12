package engine

import (
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// AddLog добавляет лог в историю инстанса
func (i *Instance) AddLog(text, logType string) {
	i.Logs = append(i.Logs, api.LogEntry{
		ID:        fmt.Sprintf("%d_%d", i.ID, time.Now().UnixNano()),
		Text:      text,
		Type:      logType,
		Timestamp: time.Now().UnixMilli(),
	})
	logger.Log.WithFields(logrus.Fields{
		"instance":  i.ID,
		"component": "game_log",
		"log_type":  logType,
	}).Info(text)
}
