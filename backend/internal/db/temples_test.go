package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type mockTempleConn struct {
	driver *mockTempleDriver
}

func (c *mockTempleConn) Prepare(query string) (driver.Stmt, error) {
	return &mockTempleStmt{conn: c, query: query}, nil
}

func (c *mockTempleConn) Close() error {
	return nil
}

func (c *mockTempleConn) Begin() (driver.Tx, error) {
	return &mockTempleTx{}, nil
}

type mockTempleTx struct{}

func (tx *mockTempleTx) Commit() error   { return nil }
func (tx *mockTempleTx) Rollback() error { return nil }

type mockTempleStmt struct {
	conn  *mockTempleConn
	query string
}

func (s *mockTempleStmt) Close() error { return nil }

func (s *mockTempleStmt) NumInput() int { return -1 }

func (s *mockTempleStmt) Exec(args []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}

func (s *mockTempleStmt) Query(args []driver.Value) (driver.Rows, error) {
	d := s.conn.driver
	d.mu.Lock()
	defer d.mu.Unlock()

	cols := []string{
		"id", "temple_id", "vip_puja_id", "user_id", "devotee_name",
		"devotee_gotram", "devotee_nakshatram", "devotee_phone",
		"visit_date", "slot_time", "ticket_type", "qr_token",
		"status", "amount_paid_minor", "amount_paid",
		"payment_mode", "check_in_timestamp", "created_at",
	}

	if strings.Contains(s.query, "UPDATE") {
		// Atomic update check: WHERE qr_token = $1 AND status <> 'CHECKED_IN'
		if d.status != "CHECKED_IN" {
			d.status = "CHECKED_IN"
			atomic.AddInt32(&d.successUpdates, 1)
			now := time.Now()
			return &mockTempleRows{
				columns: cols,
				values: [][]driver.Value{
					{
						"ticket-uuid-1234", "temple-uuid-1", "vip-puja-1", "devotee-uuid-1", "Devotee Test",
						"Kashyapa", "Rohini", "+919876543210",
						now, "09:00", "VIP", "VALID_QR_123",
						"CHECKED_IN", int64(50000), float64(500.0),
						"UPI", &now, now,
					},
				},
			}, nil
		}
		// 0 rows updated
		return &mockTempleRows{
			columns: cols,
			values:  [][]driver.Value{},
		}, nil
	}

	// Fallback SELECT status, check_in_timestamp:
	now := time.Now()
	return &mockTempleRows{
		columns: []string{"status", "check_in_timestamp"},
		values: [][]driver.Value{
			{d.status, &now},
		},
	}, nil
}

type mockTempleRows struct {
	columns []string
	values  [][]driver.Value
	idx     int
}

func (r *mockTempleRows) Columns() []string {
	return r.columns
}

func (r *mockTempleRows) Close() error {
	return nil
}

func (r *mockTempleRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.values) {
		return io.EOF
	}
	row := r.values[r.idx]
	for i, val := range row {
		dest[i] = val
	}
	r.idx++
	return nil
}

type mockTempleDriver struct {
	mu             sync.Mutex
	status         string
	successUpdates int32
}

func (d *mockTempleDriver) Open(name string) (driver.Conn, error) {
	return &mockTempleConn{driver: d}, nil
}

func TestTempleTicketDoubleCheckinRace(t *testing.T) {
	driverInstance := &mockTempleDriver{
		status: "BOOKED",
	}
	driverName := fmt.Sprintf("mock_temple_driver_%d", time.Now().UnixNano())
	sql.Register(driverName, driverInstance)

	sqlDB, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("failed opening mock db: %v", err)
	}
	defer sqlDB.Close()

	storage := &Storage{DB: sqlDB}

	const goroutines = 20
	var wg sync.WaitGroup
	var successCount int32
	var alreadyCheckedInCount int32
	var otherErrorCount int32

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ticket, err := storage.VerifyTempleTicket("VALID_QR_123")
			if err == nil && ticket != nil {
				atomic.AddInt32(&successCount, 1)
			} else if err != nil && strings.Contains(err.Error(), "ALREADY CHECKED IN") {
				atomic.AddInt32(&alreadyCheckedInCount, 1)
			} else {
				t.Logf("unexpected error: %v", err)
				atomic.AddInt32(&otherErrorCount, 1)
			}
		}()
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("expected exactly 1 successful check-in, got: %d", successCount)
	}
	if alreadyCheckedInCount != goroutines-1 {
		t.Errorf("expected %d tickets to be rejected as ALREADY CHECKED IN, got: %d", goroutines-1, alreadyCheckedInCount)
	}
	if otherErrorCount != 0 {
		t.Errorf("expected 0 other errors, got: %d", otherErrorCount)
	}
	if driverInstance.successUpdates != 1 {
		t.Errorf("expected driver to execute exactly 1 successful atomic update, got: %d", driverInstance.successUpdates)
	}
}
