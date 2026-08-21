package platform

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wyw14/cry-054/internal/application"
)

type NotificationRecord struct {
	Sequence    int64
	RecipientID string
	Template    string
	Values      map[string]string
	DeliveredAt time.Time
}

type LocalNotifier struct {
	mu      sync.Mutex
	records []NotificationRecord
	clock   application.Clock
}

type notificationEnvelope struct {
	recipient string
	template  string
	values    map[string]string
	prepared  time.Time
}

func prepareNotification(notification application.Notification, now time.Time) (notificationEnvelope, error) {
	recipient := strings.TrimSpace(notification.RecipientID)
	template := strings.TrimSpace(notification.Template)
	if recipient == "" || template == "" {
		return notificationEnvelope{}, fmt.Errorf("notification recipient and template are required")
	}
	values := make(map[string]string, len(notification.Values))
	for key, value := range notification.Values {
		key = strings.TrimSpace(key)
		if key == "" {
			return notificationEnvelope{}, fmt.Errorf("notification value key is required")
		}
		values[key] = strings.TrimSpace(value)
	}
	return notificationEnvelope{recipient: recipient, template: template, values: values, prepared: now.UTC()}, nil
}

func NewLocalNotifier(clock application.Clock) *LocalNotifier {
	return &LocalNotifier{clock: clock, records: make([]NotificationRecord, 0)}
}

func (n *LocalNotifier) Send(ctx context.Context, notification application.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	envelope, err := prepareNotification(notification, n.clock.Now())
	if err != nil {
		return err
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.records = append(n.records, NotificationRecord{
		Sequence:    int64(len(n.records) + 1),
		RecipientID: envelope.recipient,
		Template:    envelope.template,
		Values:      envelope.values,
		DeliveredAt: envelope.prepared,
	})
	return nil
}

func (n *LocalNotifier) Records() []NotificationRecord {
	n.mu.Lock()
	defer n.mu.Unlock()
	records := append([]NotificationRecord(nil), n.records...)
	sort.SliceStable(records, func(i, j int) bool { return records[i].Sequence < records[j].Sequence })
	return records
}
