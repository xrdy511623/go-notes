package notifier_test

import (
	"strings"
	"testing"

	"go-notes/designpattern/bridge/channel"
	"go-notes/designpattern/bridge/message"
	"go-notes/designpattern/bridge/notifier"
)

func TestAlert_ViaEmail(t *testing.T) {
	email := channel.NewEmail("ops@example.com")
	n := notifier.New(email)

	alert := message.NewAlert("CPU 过载", "critical")
	if err := n.Notify("admin@example.com", alert); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := email.Sent()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].To != "admin@example.com" {
		t.Errorf("expected to=admin@example.com, got %q", records[0].To)
	}
	if records[0].Subject == "" {
		t.Error("subject should not be empty")
	}
}

func TestAlert_ViaSMS(t *testing.T) {
	sms := channel.NewSMS("https://sms-gw.example.com")
	n := notifier.New(sms)

	alert := message.NewAlert("磁盘空间不足", "warn")
	if err := n.Notify("+8613800138000", alert); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := sms.Sent()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].To != "+8613800138000" {
		t.Errorf("expected to=+8613800138000, got %q", records[0].To)
	}
}

func TestAlert_ViaWebhook(t *testing.T) {
	wh := channel.NewWebhook("https://hooks.slack.com/services/T00/B00/xxx")
	n := notifier.New(wh)

	alert := message.NewAlert("部署失败", "critical")
	if err := n.Notify("oncall-channel", alert); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := wh.Sent()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].To != "oncall-channel" {
		t.Errorf("expected to=oncall-channel, got %q", records[0].To)
	}
}

func TestMarketing_ViaEmail(t *testing.T) {
	email := channel.NewEmail("marketing@shop.com")
	n := notifier.New(email)

	promo := message.NewMarketing("双十一大促", "全场五折，限时抢购！")
	if err := n.Notify("user@example.com", promo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := email.Sent()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Subject != "双十一大促" {
		t.Errorf("expected subject=双十一大促, got %q", records[0].Subject)
	}
	if !strings.Contains(records[0].Body, "To unsubscribe, reply STOP.") {
		t.Errorf("marketing body should contain unsubscribe footer, got %q", records[0].Body)
	}
}

func TestNotifyVia_OverridesDefault(t *testing.T) {
	email := channel.NewEmail("noreply@example.com")
	sms := channel.NewSMS("https://sms-gw.example.com")
	n := notifier.New(email)

	alert := message.NewAlert("服务异常", "warn")
	if err := n.NotifyVia("+8613900139000", alert, sms); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(email.Sent()) != 0 {
		t.Error("default email channel should have no records")
	}
	if len(sms.Sent()) != 1 {
		t.Fatalf("expected 1 SMS record, got %d", len(sms.Sent()))
	}
	if sms.Sent()[0].To != "+8613900139000" {
		t.Errorf("expected to=+8613900139000, got %q", sms.Sent()[0].To)
	}
}

func TestBroadcast_MultipleRecipients(t *testing.T) {
	sms := channel.NewSMS("https://sms-gw.example.com")
	n := notifier.New(sms)

	alert := message.NewAlert("系统维护通知", "info")
	recipients := []string{"+8613800000001", "+8613800000002", "+8613800000003"}

	errs := n.Broadcast(recipients, alert)
	if errs != nil {
		t.Fatalf("expected no errors, got %v", errs)
	}

	records := sms.Sent()
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	for i, r := range records {
		if r.To != recipients[i] {
			t.Errorf("record[%d]: expected to=%s, got %s", i, recipients[i], r.To)
		}
	}
}

func TestBroadcast_PartialFailure(t *testing.T) {
	email := channel.NewEmail("alert@example.com")
	n := notifier.New(email)

	alert := message.NewAlert("紧急告警", "critical")
	recipients := []string{"good@example.com", "bad-address", "also-good@example.com"}

	errs := n.Broadcast(recipients, alert)
	if errs == nil {
		t.Fatal("expected errors map, got nil")
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if _, ok := errs["bad-address"]; !ok {
		t.Error("expected error for 'bad-address'")
	}

	records := email.Sent()
	if len(records) != 2 {
		t.Fatalf("expected 2 successful records, got %d", len(records))
	}
}

func TestAlert_SeverityFormat(t *testing.T) {
	email := channel.NewEmail("ops@example.com")

	alert := message.NewAlert("内存泄漏", "critical")
	if err := alert.Send("admin@example.com", email); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	records := email.Sent()
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	want := "[ALERT][critical]"
	if !strings.Contains(records[0].Subject, want) {
		t.Errorf("subject should contain %q, got %q", want, records[0].Subject)
	}
	if !strings.Contains(records[0].Subject, "内存泄漏") {
		t.Errorf("subject should contain title, got %q", records[0].Subject)
	}
}

func TestSend_EmptyRecipient_Error(t *testing.T) {
	sms := channel.NewSMS("https://sms-gw.example.com")

	alert := message.NewAlert("测试告警", "info")
	err := alert.Send("", sms)
	if err == nil {
		t.Fatal("expected error for empty recipient, got nil")
	}
}
