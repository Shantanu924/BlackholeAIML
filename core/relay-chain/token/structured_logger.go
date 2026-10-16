package token

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents different log levels
type LogLevel string

const (
	LogLevelDebug    LogLevel = "DEBUG"
	LogLevelInfo     LogLevel = "INFO"
	LogLevelWarning  LogLevel = "WARNING"
	LogLevelError    LogLevel = "ERROR"
	LogLevelCritical LogLevel = "CRITICAL"
)

// TokenTransactionLog represents a structured log entry for token transactions
type TokenTransactionLog struct {
	// Basic transaction info
	TransactionID string `json:"transaction_id"`
	TxHash        string `json:"tx_hash"`
	Timestamp     int64  `json:"timestamp"`
	BlockHeight   uint64 `json:"block_height,omitempty"`

	// Token info
	TokenSymbol   string `json:"token_symbol"`
	TokenName     string `json:"token_name"`
	TokenDecimals uint8  `json:"token_decimals"`

	// Transaction details
	Operation string `json:"operation"` // mint, burn, transfer, approve, transferFrom
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Amount    uint64 `json:"amount"`

	// State changes
	FromBalanceBefore uint64 `json:"from_balance_before,omitempty"`
	FromBalanceAfter  uint64 `json:"from_balance_after,omitempty"`
	ToBalanceBefore   uint64 `json:"to_balance_before,omitempty"`
	ToBalanceAfter    uint64 `json:"to_balance_after,omitempty"`
	TotalSupplyBefore uint64 `json:"total_supply_before,omitempty"`
	TotalSupplyAfter  uint64 `json:"total_supply_after,omitempty"`

	// Approval specific
	Spender         string `json:"spender,omitempty"`
	AllowanceBefore uint64 `json:"allowance_before,omitempty"`
	AllowanceAfter  uint64 `json:"allowance_after,omitempty"`

	// Gas and fees
	GasUsed        uint64 `json:"gas_used,omitempty"`
	GasPrice       uint64 `json:"gas_price,omitempty"`
	TransactionFee uint64 `json:"transaction_fee,omitempty"`

	// Status and validation
	Status           string          `json:"status"` // success, failed, pending
	ErrorMessage     string          `json:"error_message,omitempty"`
	ValidationChecks map[string]bool `json:"validation_checks,omitempty"`

	// Context and metadata
	LogLevel  LogLevel `json:"log_level"`
	NodeID    string   `json:"node_id,omitempty"`
	UserAgent string   `json:"user_agent,omitempty"`
	IPAddress string   `json:"ip_address,omitempty"`
	SessionID string   `json:"session_id,omitempty"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Performance metrics
	ProcessingTimeMs int64 `json:"processing_time_ms,omitempty"`
	QueueTimeMs      int64 `json:"queue_time_ms,omitempty"`
}

// TokenStructuredLogger handles structured logging for token operations
type TokenStructuredLogger struct {
	logFile     *os.File
	logDir      string
	mu          sync.RWMutex
	enabled     bool
	logLevel    LogLevel
	bufferSize  int
	logBuffer   []TokenTransactionLog
	flushTicker *time.Ticker
	done        chan struct{}
}

// NewTokenStructuredLogger creates a new structured logger for token operations
func NewTokenStructuredLogger(logDir string, logLevel LogLevel) (*TokenStructuredLogger, error) {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	logFileName := fmt.Sprintf("token_transactions_%s.jsonl", timestamp)
	logFilePath := filepath.Join(logDir, logFileName)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %v", err)
	}

	logger := &TokenStructuredLogger{
		logFile:     logFile,
		logDir:      logDir,
		enabled:     true,
		logLevel:    logLevel,
		bufferSize:  100,
		logBuffer:   make([]TokenTransactionLog, 0, 100),
		flushTicker: time.NewTicker(5 * time.Second),
		done:        make(chan struct{}),
	}

	// Start background flushing
	go logger.backgroundFlush()

	return logger, nil
}

// LogTokenTransaction logs a token transaction with structured data
func (tsl *TokenStructuredLogger) LogTokenTransaction(logEntry TokenTransactionLog) {
	if !tsl.enabled {
		return
	}

	tsl.mu.Lock()
	defer tsl.mu.Unlock()

	// Set timestamp if not provided
	if logEntry.Timestamp == 0 {
		logEntry.Timestamp = time.Now().Unix()
	}

	// Set default log level if not provided
	if logEntry.LogLevel == "" {
		logEntry.LogLevel = LogLevelInfo
	}

	// Add to buffer
	tsl.logBuffer = append(tsl.logBuffer, logEntry)

	// Flush if buffer is full
	if len(tsl.logBuffer) >= tsl.bufferSize {
		tsl.flushBuffer()
	}

	// Also log to console for immediate visibility
	tsl.logToConsole(logEntry)
}

// flushBuffer writes buffered logs to file
func (tsl *TokenStructuredLogger) flushBuffer() {
	if len(tsl.logBuffer) == 0 {
		return
	}

	for _, entry := range tsl.logBuffer {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			log.Printf("Error marshaling log entry: %v", err)
			continue
		}

		// Write as JSON Lines format (one JSON object per line)
		if _, err := tsl.logFile.WriteString(string(jsonData) + "\n"); err != nil {
			log.Printf("Error writing to log file: %v", err)
		}
	}

	// Sync to disk
	tsl.logFile.Sync()

	// Clear buffer
	tsl.logBuffer = tsl.logBuffer[:0]
}

// backgroundFlush periodically flushes the buffer
func (tsl *TokenStructuredLogger) backgroundFlush() {
	for {
		select {
		case <-tsl.flushTicker.C:
			tsl.mu.Lock()
			tsl.flushBuffer()
			tsl.mu.Unlock()
		case <-tsl.done:
			return
		}
	}
}

// logToConsole logs to console with emoji and formatting
func (tsl *TokenStructuredLogger) logToConsole(entry TokenTransactionLog) {
	emoji := map[LogLevel]string{
		LogLevelDebug:    "🔍",
		LogLevelInfo:     "ℹ️",
		LogLevelWarning:  "⚠️",
		LogLevelError:    "❌",
		LogLevelCritical: "🚨",
	}

	operationEmoji := map[string]string{
		"mint":         "🪙",
		"burn":         "🔥",
		"transfer":     "💸",
		"approve":      "✅",
		"transferFrom": "🔄",
	}

	statusEmoji := map[string]string{
		"success": "✅",
		"failed":  "❌",
		"pending": "⏳",
	}

	opEmoji := operationEmoji[entry.Operation]
	if opEmoji == "" {
		opEmoji = "🔄"
	}

	statEmoji := statusEmoji[entry.Status]
	if statEmoji == "" {
		statEmoji = "❓"
	}

	// Format console output
	txHashDisplay := entry.TxHash
	if len(txHashDisplay) > 8 {
		txHashDisplay = txHashDisplay[:8] + "..."
	} else if txHashDisplay == "" {
		txHashDisplay = "no-hash"
	}
	fmt.Printf("%s %s [%s] %s %s: %d %s %s→%s (%s) [%s]\n",
		emoji[entry.LogLevel],
		opEmoji,
		entry.LogLevel,
		entry.Operation,
		statEmoji,
		entry.Amount,
		entry.TokenSymbol,
		entry.From,
		entry.To,
		txHashDisplay,
		entry.Status,
	)
}

// Close closes the logger and flushes remaining logs
func (tsl *TokenStructuredLogger) Close() error {
	tsl.mu.Lock()
	defer tsl.mu.Unlock()

	// Stop background flushing
	if tsl.flushTicker != nil {
		tsl.flushTicker.Stop()
	}
	
	// Signal background routine to exit
	select {
	case <-tsl.done:
	default:
		close(tsl.done)
	}

	// Flush remaining logs
	tsl.flushBuffer()

	// Close log file
	if tsl.logFile != nil {
		err := tsl.logFile.Close()
		tsl.logFile = nil // Prevent double close or write after close
		return err
	}

	return nil
}

// SetLogLevel sets the minimum log level
func (tsl *TokenStructuredLogger) SetLogLevel(level LogLevel) {
	tsl.mu.Lock()
	defer tsl.mu.Unlock()
	tsl.logLevel = level
}

// Enable enables or disables logging
func (tsl *TokenStructuredLogger) Enable(enabled bool) {
	tsl.mu.Lock()
	defer tsl.mu.Unlock()
	tsl.enabled = enabled
}

// GetLogStats returns statistics about logged transactions
func (tsl *TokenStructuredLogger) GetLogStats() map[string]interface{} {
	tsl.mu.RLock()
	defer tsl.mu.RUnlock()

	return map[string]interface{}{
		"enabled":          tsl.enabled,
		"log_level":        tsl.logLevel,
		"buffer_size":      tsl.bufferSize,
		"buffered_entries": len(tsl.logBuffer),
		"log_directory":    tsl.logDir,
		"log_file_open":    tsl.logFile != nil,
	}
}

// CreateTokenTransactionLog creates a structured log entry from token operation
func CreateTokenTransactionLog(token *Token, operation string, from, to string, amount uint64, status string, metadata map[string]interface{}) TokenTransactionLog {
	txHash := ""
	if metadata != nil {
		if hash, exists := metadata["tx_hash"]; exists {
			txHash = hash.(string)
		}
	}

	logEntry := TokenTransactionLog{
		TransactionID: fmt.Sprintf("%s_%d", operation, time.Now().UnixNano()),
		TxHash:        txHash,
		Timestamp:     time.Now().Unix(),
		TokenSymbol:   token.Symbol,
		TokenName:     token.Name,
		TokenDecimals: token.Decimals,
		Operation:     operation,
		From:          from,
		To:            to,
		Amount:        amount,
		Status:        status,
		LogLevel:      LogLevelInfo,
		Metadata:      metadata,
		ValidationChecks: map[string]bool{
			"address_valid":      from != "" || to != "",
			"amount_positive":    amount > 0,
			"balance_sufficient": true,
		},
	}

	// Inject processing_time_ms only when metadata was provided (nil metadata means caller
	// explicitly wants no metadata — e.g. TestCreateTokenTransactionLog nil case).
	if logEntry.Metadata != nil {
		logEntry.Metadata["processing_time_ms"] = float64(time.Since(time.Unix(logEntry.Timestamp, 0)).Milliseconds())
	}

	// Add token state information from metadata (avoids re-acquiring token lock).
	if logEntry.Metadata != nil {
		if v, ok := logEntry.Metadata["to_balance_after"].(uint64); ok {
			logEntry.ToBalanceAfter = v
		}
		if v, ok := logEntry.Metadata["from_balance_after"].(uint64); ok {
			logEntry.FromBalanceAfter = v
		}
		if v, ok := logEntry.Metadata["total_supply_after"].(uint64); ok {
			logEntry.TotalSupplyAfter = v
		}
		if v, ok := logEntry.Metadata["allowance_after"].(uint64); ok {
			logEntry.AllowanceAfter = v
		}
		if v, ok := logEntry.Metadata["tx_hash"].(string); ok && logEntry.TxHash == "" {
			logEntry.TxHash = v
		}
		if v, ok := logEntry.Metadata["error"].(string); ok && v != "" {
			logEntry.ErrorMessage = v
			logEntry.LogLevel = LogLevelError
		}
	}

	return logEntry
}

// Global logger instance
var GlobalTokenLogger *TokenStructuredLogger

// InitializeGlobalTokenLogger initializes the global token logger
func InitializeGlobalTokenLogger(logDir string, logLevel LogLevel) error {
	var err error
	GlobalTokenLogger, err = NewTokenStructuredLogger(logDir, logLevel)
	if err != nil {
		return fmt.Errorf("failed to initialize global token logger: %v", err)
	}

	log.Printf("✅ Global token structured logger initialized in %s", logDir)
	return nil
}

// LogTokenOperation is a convenience function to log token operations
func LogTokenOperation(token *Token, operation string, from, to string, amount uint64, status string, metadata map[string]interface{}) {
	if GlobalTokenLogger == nil {
		log.Printf("🪙 [%s] %s: %d %s %s→%s (%s)",
			operation, token.Symbol, amount, token.Symbol, from, to, status)
		return
	}

	// Respect log level filtering — INFO ops are suppressed when level is ERROR or CRITICAL.
	if GlobalTokenLogger.logLevel == LogLevelError || GlobalTokenLogger.logLevel == LogLevelCritical {
		if status != "failed" {
			return
		}
	}

	logEntry := CreateTokenTransactionLog(token, operation, from, to, amount, status, metadata)
	GlobalTokenLogger.LogTokenTransaction(logEntry)
}
