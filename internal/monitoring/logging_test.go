// Package monitoring provides server state monitoring and logging functionality for the VPN server.
// It implements real-time monitoring of server health, client connections, system resources,
// and comprehensive logging with metrics collection and alerting capabilities.
package monitoring

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogManager(t *testing.T) {
	t.Run("should create log manager with default configuration", func(t *testing.T) {
		lm := NewLogManager()
		defer lm.Close()

		assert.NotNil(t, lm)
		assert.NotNil(t, lm.config)
		assert.NotNil(t, lm.loggers)
		assert.Equal(t, LogLevelInfo, lm.config.LogLevel)
		assert.True(t, lm.config.LogToFile)
		assert.True(t, lm.config.LogToStdout)
		assert.Equal(t, "./logs", lm.config.LogDirectory)
		assert.Equal(t, 1000, lm.config.BufferSize)
	})
}

func TestNewLogManagerWithConfig(t *testing.T) {
	t.Run("should create log manager with custom configuration", func(t *testing.T) {
		config := LogConfig{
			LogLevel:     LogLevelDebug,
			LogToFile:    false,
			LogToStdout:  true,
			BufferSize:   500,
		}

		lm := NewLogManagerWithConfig(config)
		defer lm.Close()

		assert.NotNil(t, lm)
		assert.Equal(t, LogLevelDebug, lm.config.LogLevel)
		assert.False(t, lm.config.LogToFile)
		assert.True(t, lm.config.LogToStdout)
		assert.Equal(t, 500, lm.config.BufferSize)
	})
}

func TestLogLevel_String(t *testing.T) {
	t.Run("should return correct string representations", func(t *testing.T) {
		assert.Equal(t, "TRACE", LogLevelTrace.String())
		assert.Equal(t, "DEBUG", LogLevelDebug.String())
		assert.Equal(t, "INFO", LogLevelInfo.String())
		assert.Equal(t, "WARN", LogLevelWarn.String())
		assert.Equal(t, "ERROR", LogLevelError.String())
		assert.Equal(t, "FATAL", LogLevelFatal.String())
	})

	t.Run("should return UNKNOWN for invalid log level", func(t *testing.T) {
		invalidLevel := LogLevel(999)
		assert.Equal(t, "UNKNOWN", invalidLevel.String())
	})
}

func TestLogManager_Log(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "vpn_log_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := LogConfig{
		LogLevel:     LogLevelDebug,
		LogToFile:    true,
		LogToStdout:  false, // Disable stdout for clean tests
		LogDirectory: tempDir,
		BufferSize:   100,
	}

	lm := NewLogManagerWithConfig(config)
	defer lm.Close()

	t.Run("should log messages at or above configured level", func(t *testing.T) {
		// These should be logged (at or above DEBUG level)
		lm.Log(LogLevelDebug, "Debug message", nil)
		lm.Log(LogLevelInfo, "Info message", nil)
		lm.Log(LogLevelError, "Error message", nil)

		// This should not be logged (below DEBUG level)
		lm.Log(LogLevelTrace, "Trace message", nil)

		// Check buffer contains only the logged messages
		recent := lm.GetRecentLogs(10)
		assert.Len(t, recent, 3)
		assert.Equal(t, "Debug message", recent[0].Message)
		assert.Equal(t, "Info message", recent[1].Message)
		assert.Equal(t, "Error message", recent[2].Message)
	})

	t.Run("should log messages with metadata", func(t *testing.T) {
		metadata := map[string]interface{}{
			"user_id": 123,
			"action":  "login",
		}

		lm.Log(LogLevelInfo, "User action", metadata)

		recent := lm.GetRecentLogs(1)
		assert.Len(t, recent, 1)
		assert.Equal(t, "User action", recent[0].Message)
		assert.Equal(t, metadata, recent[0].Metadata)
	})
}

func TestLogManager_ConvenienceMethods(t *testing.T) {
	config := LogConfig{
		LogLevel:     LogLevelTrace,
		LogToFile:    false,
		LogToStdout:  false,
		BufferSize:   100,
	}

	lm := NewLogManagerWithConfig(config)
	defer lm.Close()

	t.Run("should log using convenience methods", func(t *testing.T) {
		lm.LogTrace("Trace message")
		lm.LogDebug("Debug message")
		lm.LogInfo("Info message")
		lm.LogWarn("Warn message")
		lm.LogError("Error message")
		lm.LogFatal("Fatal message")

		recent := lm.GetRecentLogs(10)
		assert.Len(t, recent, 6)

		levels := []LogLevel{LogLevelTrace, LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelFatal}
		for i, entry := range recent {
			assert.Equal(t, levels[i], entry.Level)
		}
	})
}

func TestLogManager_GetRecentLogs(t *testing.T) {
	config := LogConfig{
		LogLevel:     LogLevelInfo,
		LogToFile:    false,
		LogToStdout:  false,
		BufferSize:   10,
	}

	lm := NewLogManagerWithConfig(config)
	defer lm.Close()

	t.Run("should return recent logs in correct order", func(t *testing.T) {
		// Log 5 messages
		for i := 1; i <= 5; i++ {
			lm.LogInfo(fmt.Sprintf("Message %d", i))
		}

		recent := lm.GetRecentLogs(3)
		assert.Len(t, recent, 3)
		assert.Equal(t, "Message 3", recent[0].Message)
		assert.Equal(t, "Message 4", recent[1].Message)
		assert.Equal(t, "Message 5", recent[2].Message)
	})

	t.Run("should handle count larger than buffer", func(t *testing.T) {
		recent := lm.GetRecentLogs(20)
		assert.LessOrEqual(t, len(recent), 10) // Should not exceed buffer size
	})

	t.Run("should handle negative count", func(t *testing.T) {
		recent := lm.GetRecentLogs(-1)
		assert.GreaterOrEqual(t, len(recent), 0)
	})
}

func TestLogManager_GetLogsByLevel(t *testing.T) {
	config := LogConfig{
		LogLevel:     LogLevelTrace,
		LogToFile:    false,
		LogToStdout:  false,
		BufferSize:   100,
	}

	lm := NewLogManagerWithConfig(config)
	defer lm.Close()

	t.Run("should return logs filtered by level", func(t *testing.T) {
		// Log messages at different levels
		lm.LogInfo("Info 1")
		lm.LogError("Error 1")
		lm.LogInfo("Info 2")
		lm.LogWarn("Warn 1")
		lm.LogError("Error 2")

		// Get only error logs
		errorLogs := lm.GetLogsByLevel(LogLevelError, 10)
		assert.Len(t, errorLogs, 2)
		assert.Equal(t, "Error 1", errorLogs[0].Message)
		assert.Equal(t, "Error 2", errorLogs[1].Message)

		// Get only info logs
		infoLogs := lm.GetLogsByLevel(LogLevelInfo, 10)
		assert.Len(t, infoLogs, 2)
		assert.Equal(t, "Info 1", infoLogs[0].Message)
		assert.Equal(t, "Info 2", infoLogs[1].Message)
	})

	t.Run("should respect count limit", func(t *testing.T) {
		// Get only 1 error log
		errorLogs := lm.GetLogsByLevel(LogLevelError, 1)
		assert.Len(t, errorLogs, 1)
		assert.Equal(t, "Error 2", errorLogs[0].Message) // Should be the most recent
	})
}

func TestLogManager_GetLogsSince(t *testing.T) {
	config := LogConfig{
		LogLevel:     LogLevelInfo,
		LogToFile:    false,
		LogToStdout:  false,
		BufferSize:   100,
	}

	lm := NewLogManagerWithConfig(config)
	defer lm.Close()

	t.Run("should return logs since specified time", func(t *testing.T) {
		now := time.Now()
		
		// Log a message
		lm.LogInfo("Old message")
		
		// Wait a bit
		time.Sleep(10 * time.Millisecond)
		cutoff := time.Now()
		
		// Wait a bit more
		time.Sleep(10 * time.Millisecond)
		
		// Log another message
		lm.LogInfo("New message")

		// Get logs since cutoff
		recentLogs := lm.GetLogsSince(cutoff)
		assert.Len(t, recentLogs, 1)
		assert.Equal(t, "New message", recentLogs[0].Message)

		// Get logs since beginning
		allLogs := lm.GetLogsSince(now.Add(-time.Hour))
		assert.Len(t, allLogs, 2)
	})
}

func TestLogManager_BufferManagement(t *testing.T) {
	config := LogConfig{
		LogLevel:     LogLevelInfo,
		LogToFile:    false,
		LogToStdout:  false,
		BufferSize:   3, // Small buffer for testing
	}

	lm := NewLogManagerWithConfig(config)
	defer lm.Close()

	t.Run("should maintain buffer size limit", func(t *testing.T) {
		// Log more messages than buffer size
		for i := 1; i <= 5; i++ {
			lm.LogInfo(fmt.Sprintf("Message %d", i))
		}

		recent := lm.GetRecentLogs(10)
		assert.Len(t, recent, 3) // Should only keep last 3
		assert.Equal(t, "Message 3", recent[0].Message)
		assert.Equal(t, "Message 4", recent[1].Message)
		assert.Equal(t, "Message 5", recent[2].Message)
	})
}

func TestLogManager_FileLogging(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "vpn_log_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := LogConfig{
		LogLevel:     LogLevelInfo,
		LogToFile:    true,
		LogToStdout:  false,
		LogDirectory: tempDir,
		BufferSize:   100,
	}

	lm := NewLogManagerWithConfig(config)
	defer lm.Close()

	t.Run("should create log files", func(t *testing.T) {
		lm.LogInfo("Test message")

		// Check that log file was created
		infoLogPath := filepath.Join(tempDir, "INFO.log")
		_, err := os.Stat(infoLogPath)
		assert.NoError(t, err)
	})

	t.Run("should write to appropriate log files", func(t *testing.T) {
		lm.LogError("Error message")

		// Check that error log file was created
		errorLogPath := filepath.Join(tempDir, "ERROR.log")
		_, err := os.Stat(errorLogPath)
		assert.NoError(t, err)

		// Read file contents
		content, err := os.ReadFile(errorLogPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Error message")
	})
}

func TestLogManager_UpdateConfig(t *testing.T) {
	lm := NewLogManager()
	defer lm.Close()

	t.Run("should update configuration", func(t *testing.T) {
		newConfig := LogConfig{
			LogLevel:     LogLevelError,
			LogToFile:    false,
			LogToStdout:  true,
			BufferSize:   500,
		}

		err := lm.UpdateConfig(newConfig)
		assert.NoError(t, err)

		config := lm.GetConfig()
		assert.Equal(t, LogLevelError, config.LogLevel)
		assert.False(t, config.LogToFile)
		assert.True(t, config.LogToStdout)
		assert.Equal(t, 500, config.BufferSize)
	})
}

func TestLogManager_FormatMessage(t *testing.T) {
	lm := NewLogManager()
	defer lm.Close()

	t.Run("should format message without metadata", func(t *testing.T) {
		entry := LogEntry{
			Message: "Simple message",
		}

		formatted := lm.formatMessage(entry)
		assert.Equal(t, "Simple message", formatted)
	})

	t.Run("should format message with metadata", func(t *testing.T) {
		entry := LogEntry{
			Message: "Message with metadata",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
		}

		formatted := lm.formatMessage(entry)
		assert.Contains(t, formatted, "Message with metadata |")
		assert.Contains(t, formatted, "key1=value1")
		assert.Contains(t, formatted, "key2=123")
	})
}

func TestLogManager_Close(t *testing.T) {
	// Create temporary directory for test logs
	tempDir, err := os.MkdirTemp("", "vpn_log_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := LogConfig{
		LogLevel:     LogLevelInfo,
		LogToFile:    true,
		LogToStdout:  false,
		LogDirectory: tempDir,
		BufferSize:   100,
	}

	lm := NewLogManagerWithConfig(config)

	t.Run("should close log files without error", func(t *testing.T) {
		// Log something to create files
		lm.LogInfo("Test message")

		// Close should not return error
		err := lm.Close()
		assert.NoError(t, err)
	})
}