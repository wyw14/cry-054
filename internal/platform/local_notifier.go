package platform

import (
	"context"
	"fmt"
	"sort"
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

func NewLocalNotifier(clock application.Clock) *LocalNotifier {
	return &LocalNotifier{clock: clock, records: make([]NotificationRecord, 0)}
}

func (n *LocalNotifier) Send(ctx context.Context, notification application.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if notification.RecipientID == "" || notification.Template == "" {
		return fmt.Errorf("notification recipient and template are required")
	}
	values := make(map[string]string, len(notification.Values))
	for key, value := range notification.Values {
		values[key] = value
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.records = append(n.records, NotificationRecord{
		Sequence:    int64(len(n.records) + 1),
		RecipientID: notification.RecipientID,
		Template:    notification.Template,
		Values:      values,
		DeliveredAt: n.clock.Now().UTC(),
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
