package db

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/purohit-platform/backend/internal/models"
)

type mockBookingTx struct {
	driver *mockBookingDriver
}

func (tx *mockBookingTx) Commit() error {
	atomic.AddInt32(&tx.driver.commits, 1)
	return nil
}

func (tx *mockBookingTx) Rollback() error {
	atomic.AddInt32(&tx.driver.rollbacks, 1)
	return nil
}

type mockBookingConn struct {
	driver *mockBookingDriver
}

func (c *mockBookingConn) Prepare(query string) (driver.Stmt, error) {
	return &mockBookingStmt{conn: c, query: query}, nil
}

func (c *mockBookingConn) Close() error { return nil }

func (c *mockBookingConn) Begin() (driver.Tx, error) {
	atomic.AddInt32(&c.driver.begins, 1)
	return &mockBookingTx{driver: c.driver}, nil
}

type mockBookingStmt struct {
	conn  *mockBookingConn
	query string
}

func (s *mockBookingStmt) Close() error { return nil }
func (s *mockBookingStmt) NumInput() int { return -1 }

func (s *mockBookingStmt) Exec(args []driver.Value) (driver.Result, error) {
	// If inserting into booking_materials, force a simulated failure
	if strings.Contains(s.query, "booking_materials") && s.conn.driver.failMaterials {
		return nil, errors.New("simulated foreign key / material insert failure")
	}
	return driver.RowsAffected(1), nil
}

func (s *mockBookingStmt) Query(args []driver.Value) (driver.Rows, error) {
	if strings.Contains(s.query, "FROM users") {
		return &mockBookingRows{
			columns: []string{"id"},
			values: [][]driver.Value{
				{"u1111111-2f3b-485a-8b1a-b6058e469599"},
			},
		}, nil
	}

	if strings.Contains(s.query, "INSERT INTO bookings") {
		return &mockBookingRows{
			columns: []string{"id"},
			values: [][]driver.Value{
				{"b1111111-2f3b-485a-8b1a-b6058e469599"},
			},
		}, nil
	}

	if strings.Contains(s.query, "pooja_samagri") {
		return &mockBookingRows{
			columns: []string{"id", "pooja_id", "item_name", "category", "quantity", "is_mandatory", "estimated_price", "estimated_price_minor"},
			values: [][]driver.Value{
				{"item-1", "pooja-1", "Kumkum", "Sacred Ritual Samagri", "50g", true, float64(10.0), int64(1000)},
			},
		}, nil
	}

	return &mockBookingRows{
		columns: []string{"id"},
		values:  [][]driver.Value{},
	}, nil
}

type mockBookingRows struct {
	columns []string
	values  [][]driver.Value
	idx     int
}

func (r *mockBookingRows) Columns() []string { return r.columns }
func (r *mockBookingRows) Close() error       { return nil }
func (r *mockBookingRows) Next(dest []driver.Value) error {
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

type mockBookingDriver struct {
	failMaterials bool
	begins        int32
	commits       int32
	rollbacks     int32
}

func (d *mockBookingDriver) Open(name string) (driver.Conn, error) {
	return &mockBookingConn{driver: d}, nil
}

func TestCreateBookingAtomicRollbackOnMaterialError(t *testing.T) {
	driverInstance := &mockBookingDriver{
		failMaterials: true, // Force material insertion failure
	}
	driverName := fmt.Sprintf("mock_booking_driver_%d", time.Now().UnixNano())
	sql.Register(driverName, driverInstance)

	sqlDB, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("failed opening mock db: %v", err)
	}
	defer sqlDB.Close()

	storage := &Storage{DB: sqlDB}

	req := models.BookingCreateRequest{
		UserID:            "devotee-1",
		PriestID:          "priest-1",
		PoojaID:           "pooja-1",
		BookingType:       "DIRECT",
		IncludeSamagriKit: true,
		TotalAmountMinor:  250000,
	}

	bookingID, otp, err := storage.CreateBooking(req)

	// Assert that creation failed
	if err == nil {
		t.Fatalf("expected CreateBooking to fail on material error, but got bookingID=%s, otp=%s", bookingID, otp)
	}

	if !strings.Contains(err.Error(), "failed to insert booking material") {
		t.Errorf("expected 'failed to insert booking material' error, got: %v", err)
	}

	// Assert atomic transaction behavior: transaction began and rolled back, never committed
	if driverInstance.begins != 1 {
		t.Errorf("expected 1 transaction begin, got: %d", driverInstance.begins)
	}
	if driverInstance.rollbacks != 1 {
		t.Errorf("expected 1 transaction rollback, got: %d", driverInstance.rollbacks)
	}
	if driverInstance.commits != 0 {
		t.Errorf("expected 0 commits on failure, got: %d", driverInstance.commits)
	}
}
