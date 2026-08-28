package store

import (
	"context"
	"testing"
)

func TestLogNotification(t *testing.T) {
	s := openTemp(t)
	ctx := context.Background()
	site := mustSite(t, s)
	run, _ := s.CreateRun(ctx, sampleRun(site.ID))
	ch, _ := s.CreateChannel(ctx, Channel{Type: ChannelShoutrrr, Name: "C", ConfigEncrypted: []byte{1}})
	if err := s.LogNotification(ctx, NotificationLog{RunID: run.ID, ChannelID: ch.ID, Status: "sent"}); err != nil {
		t.Fatalf("LogNotification: %v", err)
	}
	n, err := s.CountNotificationLogs(ctx, run.ID)
	if err != nil {
		t.Fatalf("CountNotificationLogs: %v", err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
}
