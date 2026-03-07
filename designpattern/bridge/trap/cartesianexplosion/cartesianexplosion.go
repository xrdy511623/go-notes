// Package cartesianexplosion 展示不使用桥接模式时的笛卡尔积爆炸问题。
//
// 假设有 M 种消息类型、N 种投递渠道:
//   - 不用桥接: 需要 M × N 个具体类型（本例 2×3 = 6）
//   - 使用桥接: 只需要 M + N 个类型（本例 2+3 = 5）
//
// 每新增一种消息类型，必须同时新增 N 个结构体；
// 每新增一种渠道，必须同时新增 M 个结构体。
// 投递逻辑和格式化逻辑在每个结构体中被重复实现，违反 DRY 原则。
package cartesianexplosion

// ---------------------------------------------------------------------------
// 2 种消息 × 3 种渠道 = 6 个具体类型
// ---------------------------------------------------------------------------

// AlertEmailNotifier 告警 + 邮件
type AlertEmailNotifier struct{ from string }

func (n *AlertEmailNotifier) Send(to string) error { return nil }

// AlertSMSNotifier 告警 + 短信
type AlertSMSNotifier struct{ gateway string }

func (n *AlertSMSNotifier) Send(to string) error { return nil }

// AlertWebhookNotifier 告警 + Webhook
type AlertWebhookNotifier struct{ url string }

func (n *AlertWebhookNotifier) Send(to string) error { return nil }

// MarketingEmailNotifier 营销 + 邮件
type MarketingEmailNotifier struct{ from string }

func (n *MarketingEmailNotifier) Send(to string) error { return nil }

// MarketingSMSNotifier 营销 + 短信
type MarketingSMSNotifier struct{ gateway string }

func (n *MarketingSMSNotifier) Send(to string) error { return nil }

// MarketingWebhookNotifier 营销 + Webhook
type MarketingWebhookNotifier struct{ url string }

func (n *MarketingWebhookNotifier) Send(to string) error { return nil }

// ---------------------------------------------------------------------------
// 如果新增 "VerificationMessage" 类型，需要再加 3 个结构体:
//   VerificationEmailNotifier
//   VerificationSMSNotifier
//   VerificationWebhookNotifier
//
// 如果新增 "PushNotification" 渠道，需要再加 M 个结构体:
//   AlertPushNotifier
//   MarketingPushNotifier
//   VerificationPushNotifier (如果已经有 Verification)
//
// 类型数量的增长是乘法级别的，维护成本呈指数式上升。
// ---------------------------------------------------------------------------
