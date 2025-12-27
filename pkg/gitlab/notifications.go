package gitlab

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// NotificationLevel defines the severity of a notification
type NotificationLevel int

const (
	NotificationInfo NotificationLevel = iota
	NotificationWarning
	NotificationError
)

// String returns the string representation of the notification level
func (n NotificationLevel) String() string {
	switch n {
	case NotificationInfo:
		return "INFO"
	case NotificationWarning:
		return "WARNING"
	case NotificationError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Notification represents a message to both AI and user
type Notification struct {
	Level     NotificationLevel `json:"level"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	TokenName string            `json:"tokenName,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// notificationStore holds recent notifications in memory
var notificationStore = make([]Notification, 0, 100)

// SendNotification sends a notification to the user (via stderr) and stores it for AI retrieval
func SendNotification(logger *log.Logger, notif Notification) {
	notif.Timestamp = time.Now()

	// Log to stderr (visible to user)
	msg := fmt.Sprintf("[%s] %s: %s",
		notif.Level.String(),
		notif.Title,
		notif.Message)

	switch notif.Level {
	case NotificationError:
		logger.Error(msg)
	case NotificationWarning:
		logger.Warn(msg)
	default:
		logger.Info(msg)
	}

	// Store in memory for AI to retrieve
	// Keep only last 100 notifications
	notificationStore = append(notificationStore, notif)
	if len(notificationStore) > 100 {
		notificationStore = notificationStore[1:]
	}
}

// GetNotifications returns all stored notifications
func GetNotifications() []Notification {
	return notificationStore
}

// ClearNotifications clears all stored notifications
func ClearNotifications() {
	notificationStore = make([]Notification, 0, 100)
}

// notifyTokenIssue sends a notification about token problems
func notifyTokenIssue(logger *log.Logger, tokenName string, err error) {
	SendNotification(logger, Notification{
		Level:     NotificationWarning,
		Title:     "Token Issue Detected",
		Message:   fmt.Sprintf("Token '%s' has a problem: %v", tokenName, err),
		TokenName: tokenName,
	})
}

// notifyTokenExpiration sends a notification about expired token
func notifyTokenExpiration(logger *log.Logger, tokenName string) {
	SendNotification(logger, Notification{
		Level:     NotificationError,
		Title:     "Token Expired",
		Message:   fmt.Sprintf("Token '%s' is expired or invalid. Please update it using the updateToken tool or reconfigure the MCP server.", tokenName),
		TokenName: tokenName,
	})
}

// notifyTokenExpiringSoon sends a warning about token expiring soon
func notifyTokenExpiringSoon(logger *log.Logger, tokenName string, daysUntilExpiry int) {
	SendNotification(logger, Notification{
		Level:     NotificationWarning,
		Title:     "Token Expiring Soon",
		Message:   fmt.Sprintf("Token '%s' will expire in %d days. Please create a new token and update it.", tokenName, daysUntilExpiry),
		TokenName: tokenName,
	})
}

// notifyTokenValidated sends a success message about token validation
func notifyTokenValidated(logger *log.Logger, tokenName string, userID int, username string) {
	SendNotification(logger, Notification{
		Level:     NotificationInfo,
		Title:     "Token Validated",
		Message:   fmt.Sprintf("Token '%s' validated successfully for user %s (ID: %d)", tokenName, username, userID),
		TokenName: tokenName,
	})
}
