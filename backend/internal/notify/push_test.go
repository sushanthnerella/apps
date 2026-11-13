package notify

import (
	"context"
	"testing"
)

func TestLoggingNoOpNotifier(t *testing.T) {
	notifier := &LoggingNoOpNotifier{}
	msg := NotificationMessage{
		Title: "Test Title",
		Body:  "Test Body",
	}
	err := notifier.SendToToken(context.Background(), "fake-device-token", msg)
	if err != nil {
		t.Fatalf("expected nil error from no-op notifier, got: %v", err)
	}

	err = notifier.SendToTokens(context.Background(), []string{"tok-1", "tok-2"}, msg)
	if err != nil {
		t.Fatalf("expected nil error from no-op SendToTokens, got: %v", err)
	}
}

func TestNewPushNotifierFromEnv_Unset(t *testing.T) {
	notifier := NewPushNotifierFromEnv()
	if notifier == nil {
		t.Fatal("expected non-nil notifier from env")
	}
	if _, ok := notifier.(*LoggingNoOpNotifier); !ok {
		t.Fatalf("expected LoggingNoOpNotifier when FCM_SERVICE_ACCOUNT_JSON is unset, got %T", notifier)
	}
}
