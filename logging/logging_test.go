package logging

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

type myTestyLogger struct {
	mock.Mock
	wg sync.WaitGroup
}

func (l *myTestyLogger) Printf(format string, v ...interface{}) {
	l.Called(format, v)
	l.wg.Done()
}

func TestRecordLogsDebug(t *testing.T) {
	logger := &myTestyLogger{}
	logger.wg.Add(2)
	logger.On("Printf", logFormat, []interface{}{'I', "test1", "info message"}).Once()
	logger.On("Printf", logFormat, []interface{}{'D', "test1", "debug message"}).Once()

	logCh = make(chan logEntry)
	go RecordLogs(logger, true)

	Debug("test1", "debug message")
	Log("test1", "info message")

	// Timeout
	go func() {
		time.Sleep(500 * time.Millisecond)
		logger.wg.Done()
		logger.wg.Done()
	}()
	logger.wg.Wait()
	close(logCh)
}

func TestRecordLogsNoDebug(t *testing.T) {
	logger := &myTestyLogger{}
	logger.wg.Add(1)
	logger.On("Printf", logFormat, []interface{}{'I', "test2", "info message"}).Once()

	logCh = make(chan logEntry)
	go RecordLogs(logger, false)

	Debug("test2", "debug message")
	Log("test2", "info message")

	// Timeout
	go func() {
		time.Sleep(500 * time.Millisecond)
		logger.wg.Done()
		logger.wg.Done()
	}()
	logger.wg.Wait()
	close(logCh)
}
