// Package monitoring provides server state monitoring and logging functionality for the VPN server.
// It implements real-time monitoring of server health, client connections, system resources,
// and comprehensive logging with metrics collection and alerting capabilities.
package monitoring

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogManager manages logging operations for the VPN server monitoring system.
// It provides structured logging with different log levels, file rotation,
// and configurable output destinations for comprehensive log management.
type LogManager struct {
	config     LogConfig          // Logging configuration
	loggers    map[LogLevel]*log.Logger // Loggers for different levels
	logFiles   map[LogLevel]*os.File    // Log file handles
	mutex      sync.RWMutex       // Mutex for thread-safe operations
	logBuffer  []LogEntry         // Buffer for recent log entries
	bufferSize int                // Maximum buffer size
}

// LogConfig represents configuration options for the logging system.
type LogConfig struct {
	LogLevel        LogLevel `json:"log_level"`        // Minimum log level to record
	LogToFile       bool     `json:"log_to_file"`      // Whether to write logs to file
	LogToStdout     bool     `json:"log_to_stdout"`    // Whether to write logs to stdout
	LogDirectory    string   `json:"log_directory"`    // Directory for log files
	MaxFileSize     int64    `json:"max_file_size"`    // Maximum log file size in bytes
	MaxFiles        int      `json:"max_files"`        // Maximum number of log files to keep
	CompressOldLogs bool     `json:"compress_old_logs"` // Whether to compress rotated logs
	IncludeSource   bool     `json:"include_source"`   // Whether to include source file/line
	BufferSize      int      `json:"buffer_size"`      // Number of recent logs to keep in memory
}

// LogLevel represents the severity level of a log entry.
type LogLevel int

const (
	LogLevelTrace LogLevel = iota // Trace level - very detailed debugging
	LogLevelDebug                 // Debug level - debugging information
	LogLevelInfo                  // Info level - general information
	LogLevelWarn                  // Warn level - warning messages
	LogLevelError                 // Error level - error conditions
	LogLevelFatal                 // Fatal level - fatal errors
)

// String returns the string representation of a log level.
func (ll LogLevel) String() string {
	switch ll {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry with metadata.
type LogEntry struct {
	Timestamp time.Time   `json:"timestamp"` // When the log entry was created
	Level     LogLevel    `json:"level"`     // Log level of the entry
	Message   string      `json:"message"`   // Log message content
	Source    string      `json:"source"`    // Source file and line (if enabled)
	Component string      `json:"component"` // Component that generated the log
	Metadata  map[string]interface{} `json:"metadata"` // Additional metadata
}

// NewLogManager creates a new log manager with default configuration.
// It initializes logging with sensible defaults for production use,
// including file logging and appropriate log levels.
// Returns a pointer to the newly created LogManager.
func NewLogManager() *LogManager {
	config := LogConfig{
		LogLevel:        LogLevelInfo,
		LogToFile:       true,
		LogToStdout:     true,
		LogDirectory:    "./logs",
		MaxFileSize:     100 * 1024 * 1024, // 100MB
		MaxFiles:        10,
		CompressOldLogs: true,
		IncludeSource:   false,
		BufferSize:      1000,
	}

	manager := &LogManager{
		config:     config,
		loggers:    make(map[LogLevel]*log.Logger),
		logFiles:   make(map[LogLevel]*os.File),
		logBuffer:  make([]LogEntry, 0, config.BufferSize),
		bufferSize: config.BufferSize,
	}

	manager.initializeLoggers()
	return manager
}

// NewLogManagerWithConfig creates a new log manager with custom configuration.
// This allows fine-tuning of logging behavior for specific deployment requirements.
// Returns a pointer to the newly created LogManager.
func NewLogManagerWithConfig(config LogConfig) *LogManager {
	manager := &LogManager{
		config:     config,
		loggers:    make(map[LogLevel]*log.Logger),
		logFiles:   make(map[LogLevel]*os.File),
		logBuffer:  make([]LogEntry, 0, config.BufferSize),
		bufferSize: config.BufferSize,
	}

	manager.initializeLoggers()
	return manager
}

// initializeLoggers sets up loggers for different log levels.
func (lm *LogManager) initializeLoggers() {
	// Create log directory if it doesn't exist
	if lm.config.LogToFile {
		if err := os.MkdirAll(lm.config.LogDirectory, 0755); err != nil {
			log.Printf("Failed to create log directory: %v", err)
			lm.config.LogToFile = false
		}
	}

	// Initialize loggers for each level
	levels := []LogLevel{LogLevelTrace, LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelFatal}
	
	for _, level := range levels {
		var writers []io.Writer

		// Add stdout writer if enabled
		if lm.config.LogToStdout {
			writers = append(writers, os.Stdout)
		}

		// Add file writer if enabled
		if lm.config.LogToFile {
			filename := filepath.Join(lm.config.LogDirectory, fmt.Sprintf("%s.log", level.String()))
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				log.Printf("Failed to open log file %s: %v", filename, err)
			} else {
				writers = append(writers, file)
				lm.logFiles[level] = file
			}
		}

		// Create multi-writer if we have multiple outputs
		var writer io.Writer
		if len(writers) == 1 {
			writer = writers[0]
		} else if len(writers) > 1 {
			writer = io.MultiWriter(writers...)
		} else {
			writer = io.Discard
		}

		// Create logger with appropriate flags
		flags := log.LstdFlags
		if lm.config.IncludeSource {
			flags |= log.Lshortfile
		}

		lm.loggers[level] = log.New(writer, fmt.Sprintf("[%s] ", level.String()), flags)
	}
}

// Log writes a log entry with the specified level and message.
// This is the main logging method that handles formatting, filtering,
// and routing log messages to appropriate destinations.
func (lm *LogManager) Log(level LogLevel, message string, metadata map[string]interface{}) {
	// Check if this log level should be recorded
	if level < lm.config.LogLevel {
		return
	}

	// Create log entry
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Component: "vpn-server",
		Metadata:  metadata,
	}

	// Add to buffer
	lm.addToBuffer(entry)

	// Get the appropriate logger
	logger, exists := lm.loggers[level]
	if !exists {
		logger = lm.loggers[LogLevelInfo] // Fallback to info logger
	}

	// Format and write the log message
	formattedMessage := lm.formatMessage(entry)
	logger.Print(formattedMessage)
}

// LogTrace logs a trace-level message.
func (lm *LogManager) LogTrace(message string) {
	lm.Log(LogLevelTrace, message, nil)
}

// LogDebug logs a debug-level message.
func (lm *LogManager) LogDebug(message string) {
	lm.Log(LogLevelDebug, message, nil)
}

// LogInfo logs an info-level message.
func (lm *LogManager) LogInfo(message string) {
	lm.Log(LogLevelInfo, message, nil)
}

// LogWarn logs a warning-level message.
func (lm *LogManager) LogWarn(message string) {
	lm.Log(LogLevelWarn, message, nil)
}

// LogError logs an error-level message.
func (lm *LogManager) LogError(message string) {
	lm.Log(LogLevelError, message, nil)
}

// LogFatal logs a fatal-level message.
func (lm *LogManager) LogFatal(message string) {
	lm.Log(LogLevelFatal, message, nil)
}

// LogWithMetadata logs a message with additional metadata.
func (lm *LogManager) LogWithMetadata(level LogLevel, message string, metadata map[string]interface{}) {
	lm.Log(level, message, metadata)
}

// GetRecentLogs returns recent log entries from the in-memory buffer.
// This is useful for displaying recent logs in dashboards or APIs.
func (lm *LogManager) GetRecentLogs(count int) []LogEntry {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	if count <= 0 || count > len(lm.logBuffer) {
		count = len(lm.logBuffer)
	}

	// Return the most recent entries
	start := len(lm.logBuffer) - count
	result := make([]LogEntry, count)
	copy(result, lm.logBuffer[start:])

	return result
}

// GetLogsByLevel returns recent log entries filtered by level.
func (lm *LogManager) GetLogsByLevel(level LogLevel, count int) []LogEntry {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	var filtered []LogEntry
	for i := len(lm.logBuffer) - 1; i >= 0 && len(filtered) < count; i-- {
		if lm.logBuffer[i].Level == level {
			filtered = append([]LogEntry{lm.logBuffer[i]}, filtered...)
		}
	}

	return filtered
}

// GetLogsSince returns log entries created after the specified time.
func (lm *LogManager) GetLogsSince(since time.Time) []LogEntry {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	var result []LogEntry
	for _, entry := range lm.logBuffer {
		if entry.Timestamp.After(since) {
			result = append(result, entry)
		}
	}

	return result
}

// RotateLogs rotates log files when they exceed the maximum size.
// This prevents log files from growing too large and manages disk space.
func (lm *LogManager) RotateLogs() error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	for level, file := range lm.logFiles {
		if file == nil {
			continue
		}

		// Check file size
		stat, err := file.Stat()
		if err != nil {
			continue
		}

		if stat.Size() > lm.config.MaxFileSize {
			// Close current file
			file.Close()

			// Rotate files
			if err := lm.rotateFile(level); err != nil {
				return fmt.Errorf("failed to rotate log file for level %s: %w", level.String(), err)
			}

			// Reopen file
			filename := filepath.Join(lm.config.LogDirectory, fmt.Sprintf("%s.log", level.String()))
			newFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return fmt.Errorf("failed to reopen log file: %w", err)
			}

			lm.logFiles[level] = newFile

			// Update logger
			var writers []io.Writer
			if lm.config.LogToStdout {
				writers = append(writers, os.Stdout)
			}
			writers = append(writers, newFile)

			var writer io.Writer
			if len(writers) == 1 {
				writer = writers[0]
			} else {
				writer = io.MultiWriter(writers...)
			}

			flags := log.LstdFlags
			if lm.config.IncludeSource {
				flags |= log.Lshortfile
			}

			lm.loggers[level] = log.New(writer, fmt.Sprintf("[%s] ", level.String()), flags)
		}
	}

	return nil
}

// Close gracefully closes all log files and cleans up resources.
func (lm *LogManager) Close() error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	for level, file := range lm.logFiles {
		if file != nil {
			if err := file.Close(); err != nil {
				return fmt.Errorf("failed to close log file for level %s: %w", level.String(), err)
			}
		}
	}

	return nil
}

// addToBuffer adds a log entry to the in-memory buffer.
func (lm *LogManager) addToBuffer(entry LogEntry) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	// Add entry to buffer
	lm.logBuffer = append(lm.logBuffer, entry)

	// Trim buffer if it exceeds maximum size
	if len(lm.logBuffer) > lm.bufferSize {
		// Remove oldest entries
		copy(lm.logBuffer, lm.logBuffer[len(lm.logBuffer)-lm.bufferSize:])
		lm.logBuffer = lm.logBuffer[:lm.bufferSize]
	}
}

// formatMessage formats a log entry into a readable string.
func (lm *LogManager) formatMessage(entry LogEntry) string {
	message := entry.Message

	// Add metadata if present
	if len(entry.Metadata) > 0 {
		message += " |"
		for key, value := range entry.Metadata {
			message += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	return message
}

// rotateFile rotates a log file by renaming it with a timestamp.
func (lm *LogManager) rotateFile(level LogLevel) error {
	originalPath := filepath.Join(lm.config.LogDirectory, fmt.Sprintf("%s.log", level.String()))
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := filepath.Join(lm.config.LogDirectory, fmt.Sprintf("%s.log.%s", level.String(), timestamp))

	// Rename current file
	if err := os.Rename(originalPath, rotatedPath); err != nil {
		return err
	}

	// Compress if enabled
	if lm.config.CompressOldLogs {
		// This would implement compression logic
		// For now, it's a placeholder
	}

	// Clean up old files
	return lm.cleanupOldLogFiles(level)
}

// cleanupOldLogFiles removes old log files exceeding the retention limit.
func (lm *LogManager) cleanupOldLogFiles(level LogLevel) error {
	pattern := filepath.Join(lm.config.LogDirectory, fmt.Sprintf("%s.log.*", level.String()))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// If we have more files than the limit, remove the oldest
	if len(matches) > lm.config.MaxFiles {
		// Sort by modification time and remove oldest
		// This is a simplified implementation
		for i := 0; i < len(matches)-lm.config.MaxFiles; i++ {
			if err := os.Remove(matches[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetConfig returns the current logging configuration.
func (lm *LogManager) GetConfig() LogConfig {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()
	
	return lm.config
}

// UpdateConfig updates the logging configuration.
// This allows dynamic reconfiguration of logging behavior.
func (lm *LogManager) UpdateConfig(config LogConfig) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	// Close existing files
	for _, file := range lm.logFiles {
		if file != nil {
			file.Close()
		}
	}

	// Update configuration
	lm.config = config
	lm.bufferSize = config.BufferSize

	// Reinitialize loggers
	lm.loggers = make(map[LogLevel]*log.Logger)
	lm.logFiles = make(map[LogLevel]*os.File)
	lm.initializeLoggers()

	return nil
}