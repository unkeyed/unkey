package testdriver

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/go-sql-driver/mysql"
)

// MockDriver wraps the MySQL driver and allows injecting failures
type MockDriver struct {
	realDriver driver.Driver
	config     *MockConfig
}

// MockConfig controls the mock driver behavior
type MockConfig struct {
	mu                 sync.RWMutex
	failNextN          int32         // Fail the next N connection attempts
	failWithError      error         // Error to return when failing
	failNextNQueries   int32         // Fail the next N queries
	failQueriesWithErr error         // Error for query failures
	connectionAttempts atomic.Int32  // Track total connection attempts
	connections        atomic.Int32  // Track active connections
	queryAttempts      atomic.Int32  // Track query attempts
}

var (
	mockDriver *MockDriver
	once       sync.Once
)

// Register registers the mock MySQL driver
func Register() *MockConfig {
	config := &MockConfig{}
	once.Do(func() {
		mockDriver = &MockDriver{
			realDriver: &mysql.MySQLDriver{},
			config:     config,
		}
		sql.Register("mysql-mock", mockDriver)
	})
	return config
}

// Open opens a new connection, possibly injecting failures
func (d *MockDriver) Open(name string) (driver.Conn, error) {
	d.config.connectionAttempts.Add(1)

	// Check if we should fail this connection
	if atomic.LoadInt32(&d.config.failNextN) > 0 {
		atomic.AddInt32(&d.config.failNextN, -1)

		d.config.mu.RLock()
		err := d.config.failWithError
		d.config.mu.RUnlock()

		if err != nil {
			return nil, err
		}
		return nil, errors.New("mock connection failure")
	}

	// Open real connection using the real MySQL driver
	conn, err := d.realDriver.Open(name)
	if err != nil {
		return nil, err
	}

	d.config.connections.Add(1)
	return &mockConn{
		realConn: conn,
		config:   d.config,
		onClose: func() {
			d.config.connections.Add(-1)
		},
	}, nil
}

// mockConn wraps a real connection
type mockConn struct {
	realConn driver.Conn
	config   *MockConfig
	onClose  func()
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	c.config.queryAttempts.Add(1)

	// Check if we should fail this query
	if atomic.LoadInt32(&c.config.failNextNQueries) > 0 {
		atomic.AddInt32(&c.config.failNextNQueries, -1)

		c.config.mu.RLock()
		err := c.config.failQueriesWithErr
		c.config.mu.RUnlock()

		if err != nil {
			return nil, err
		}
		return nil, errors.New("mock query failure")
	}

	return c.realConn.Prepare(query)
}

func (c *mockConn) Close() error {
	if c.onClose != nil {
		c.onClose()
	}
	return c.realConn.Close()
}

func (c *mockConn) Begin() (driver.Tx, error) {
	return c.realConn.Begin()
}

// Config methods

// FailNext makes the next N connection attempts fail
func (c *MockConfig) FailNext(n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.StoreInt32(&c.failNextN, int32(n))
	c.failWithError = err
}

// FailNextQueries makes the next N queries fail
func (c *MockConfig) FailNextQueries(n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.StoreInt32(&c.failNextNQueries, int32(n))
	c.failQueriesWithErr = err
}

// Reset resets all counters and failure states
func (c *MockConfig) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.StoreInt32(&c.failNextN, 0)
	atomic.StoreInt32(&c.failNextNQueries, 0)
	c.failWithError = nil
	c.failQueriesWithErr = nil
	c.connectionAttempts.Store(0)
	c.connections.Store(0)
	c.queryAttempts.Store(0)
}

// ConnectionAttempts returns the total number of connection attempts
func (c *MockConfig) ConnectionAttempts() int {
	return int(c.connectionAttempts.Load())
}

// ActiveConnections returns the number of active connections
func (c *MockConfig) ActiveConnections() int {
	return int(c.connections.Load())
}

// QueryAttempts returns the total number of query attempts
func (c *MockConfig) QueryAttempts() int {
	return int(c.queryAttempts.Load())
}
