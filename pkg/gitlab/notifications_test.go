package gitlab

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    NotificationLevel
		expected string
	}{
		{
			name:     "Info level",
			level:    NotificationInfo,
			expected: "INFO",
		},
		{
			name:     "Warning level",
			level:    NotificationWarning,
			expected: "WARNING",
		},
		{
			name:     "Error level",
			level:    NotificationError,
			expected: "ERROR",
		},
		{
			name:     "Unknown level",
			level:    NotificationLevel(99),
			expected: "UNKNOWN",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.level.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSendNotification(t *testing.T) {
	// Clear notifications before test
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel) // Only log errors to reduce noise

	tests := []struct {
		name         string
		notification Notification
		validate     func(t *testing.T, notif Notification)
	}{
		{
			name: "Info notification",
			notification: Notification{
				Level:     NotificationInfo,
				Title:     "Test Info",
				Message:   "This is an info message",
				TokenName: "test-token",
			},
			validate: func(t *testing.T, notif Notification) {
				assert.Equal(t, NotificationInfo, notif.Level)
				assert.Equal(t, "Test Info", notif.Title)
				assert.Equal(t, "This is an info message", notif.Message)
				assert.Equal(t, "test-token", notif.TokenName)
				assert.False(t, notif.Timestamp.IsZero())
			},
		},
		{
			name: "Warning notification",
			notification: Notification{
				Level:   NotificationWarning,
				Title:   "Test Warning",
				Message: "This is a warning message",
			},
			validate: func(t *testing.T, notif Notification) {
				assert.Equal(t, NotificationWarning, notif.Level)
				assert.Equal(t, "Test Warning", notif.Title)
				assert.False(t, notif.Timestamp.IsZero())
			},
		},
		{
			name: "Error notification",
			notification: Notification{
				Level:     NotificationError,
				Title:     "Test Error",
				Message:   "This is an error message",
				TokenName: "error-token",
			},
			validate: func(t *testing.T, notif Notification) {
				assert.Equal(t, NotificationError, notif.Level)
				assert.Equal(t, "Test Error", notif.Title)
				assert.Equal(t, "error-token", notif.TokenName)
				assert.False(t, notif.Timestamp.IsZero())
			},
		},
		{
			name: "Notification with all fields",
			notification: Notification{
				Level:     NotificationInfo,
				Title:     "Complete Notification",
				Message:   "All fields populated",
				TokenName: "my-token",
			},
			validate: func(t *testing.T, notif Notification) {
				assert.Equal(t, "Complete Notification", notif.Title)
				assert.Equal(t, "All fields populated", notif.Message)
				assert.Equal(t, "my-token", notif.TokenName)
				assert.WithinDuration(t, time.Now(), notif.Timestamp, 2*time.Second)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear before each test
			ClearNotifications()

			// Send notification
			SendNotification(logger, tc.notification)

			// Verify notification was stored
			notifications := GetNotifications()
			require.Len(t, notifications, 1, "Should have exactly one notification")

			if tc.validate != nil {
				tc.validate(t, notifications[0])
			}
		})
	}
}

func TestSendNotification_StoreLimit(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Send 150 notifications
	for i := 0; i < 150; i++ {
		SendNotification(logger, Notification{
			Level:   NotificationInfo,
			Title:   "Bulk Test",
			Message: "Test notification",
		})
	}

	// Should only keep last 100
	notifications := GetNotifications()
	assert.Len(t, notifications, 100, "Should keep only last 100 notifications")
}

func TestGetNotifications(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Initially empty
	notifications := GetNotifications()
	assert.Len(t, notifications, 0, "Should start with no notifications")

	// Add some notifications
	SendNotification(logger, Notification{
		Level:   NotificationInfo,
		Title:   "First",
		Message: "First message",
	})
	SendNotification(logger, Notification{
		Level:   NotificationWarning,
		Title:   "Second",
		Message: "Second message",
	})

	// Should have 2 notifications
	notifications = GetNotifications()
	assert.Len(t, notifications, 2, "Should have 2 notifications")

	// Verify order (first in, first out)
	assert.Equal(t, "First", notifications[0].Title)
	assert.Equal(t, "Second", notifications[1].Title)
}

func TestClearNotifications(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Add some notifications
	for i := 0; i < 5; i++ {
		SendNotification(logger, Notification{
			Level:   NotificationInfo,
			Title:   "Test",
			Message: "Notification",
		})
	}

	// Verify they exist
	notifications := GetNotifications()
	assert.Len(t, notifications, 5)

	// Clear them
	ClearNotifications()

	// Verify they're gone
	notifications = GetNotifications()
	assert.Len(t, notifications, 0, "All notifications should be cleared")

	// Clearing again should be safe
	ClearNotifications()
	notifications = GetNotifications()
	assert.Len(t, notifications, 0)
}

func TestNotifyTokenIssue(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	testErr := assert.AnError

	notifyTokenIssue(logger, "test-token", testErr)

	notifications := GetNotifications()
	require.Len(t, notifications, 1)

	notif := notifications[0]
	assert.Equal(t, NotificationWarning, notif.Level)
	assert.Equal(t, "Token Issue Detected", notif.Title)
	assert.Contains(t, notif.Message, "test-token")
	assert.Contains(t, notif.Message, testErr.Error())
	assert.Equal(t, "test-token", notif.TokenName)
	assert.False(t, notif.Timestamp.IsZero())
}

func TestNotifyTokenExpiration(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	notifyTokenExpiration(logger, "expired-token")

	notifications := GetNotifications()
	require.Len(t, notifications, 1)

	notif := notifications[0]
	assert.Equal(t, NotificationError, notif.Level)
	assert.Equal(t, "Token Expired", notif.Title)
	assert.Contains(t, notif.Message, "expired-token")
	assert.Contains(t, notif.Message, "updateToken")
	assert.Equal(t, "expired-token", notif.TokenName)
	assert.False(t, notif.Timestamp.IsZero())
}

func TestNotifyTokenExpiringSoon(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	notifyTokenExpiringSoon(logger, "soon-token", 7)

	notifications := GetNotifications()
	require.Len(t, notifications, 1)

	notif := notifications[0]
	assert.Equal(t, NotificationWarning, notif.Level)
	assert.Equal(t, "Token Expiring Soon", notif.Title)
	assert.Contains(t, notif.Message, "soon-token")
	assert.Contains(t, notif.Message, "7 days")
	assert.Equal(t, "soon-token", notif.TokenName)
	assert.False(t, notif.Timestamp.IsZero())
}

func TestNotifyTokenValidated(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	notifyTokenValidated(logger, "valid-token", 12345, "testuser")

	notifications := GetNotifications()
	require.Len(t, notifications, 1)

	notif := notifications[0]
	assert.Equal(t, NotificationInfo, notif.Level)
	assert.Equal(t, "Token Validated", notif.Title)
	assert.Contains(t, notif.Message, "valid-token")
	assert.Contains(t, notif.Message, "testuser")
	assert.Contains(t, notif.Message, "12345")
	assert.Equal(t, "valid-token", notif.TokenName)
	assert.False(t, notif.Timestamp.IsZero())
}

func TestNotifications_ConcurrentAccess(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Send notifications from multiple goroutines
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			for j := 0; j < 10; j++ {
				SendNotification(logger, Notification{
					Level:   NotificationInfo,
					Title:   "Concurrent Test",
					Message: "Notification",
				})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have all 100 notifications
	notifications := GetNotifications()
	assert.Len(t, notifications, 100)
}

func TestNotifications_PersistenceAcrossCalls(t *testing.T) {
	ClearNotifications()

	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	// Send a notification
	SendNotification(logger, Notification{
		Level:   NotificationInfo,
		Title:   "Persistence Test",
		Message: "Testing persistence",
	})

	// Get notifications (first call)
	notifications1 := GetNotifications()
	assert.Len(t, notifications1, 1)

	// Get notifications again (second call - should return same data)
	notifications2 := GetNotifications()
	assert.Len(t, notifications2, 1)
	assert.Equal(t, notifications1[0].Title, notifications2[0].Title)

	// Clear and verify
	ClearNotifications()
	notifications3 := GetNotifications()
	assert.Len(t, notifications3, 0)
}
