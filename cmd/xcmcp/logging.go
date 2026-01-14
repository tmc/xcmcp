package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// broadcastLog sends a logging notification to all active sessions.
// It also logs to stderr for immediate visibility.
func broadcastLog(s *mcp.Server, level mcp.LoggingLevel, logger, msg string) {
	// Log to stderr
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", timestamp, level, msg)

	// Send to MCP clients
	for session := range s.Sessions() {
		err := session.Log(context.Background(), &mcp.LoggingMessageParams{
			Level:  level,
			Logger: logger,
			Data:   msg,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG: Failed to send log to session: %v\n", err)
		}
	}
}
