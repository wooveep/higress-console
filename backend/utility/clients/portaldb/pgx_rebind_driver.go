package portaldb

import (
	"context"
	"database/sql"
	"database/sql/driver"

	"github.com/jackc/pgx/v5/stdlib"
)

const pgxRebindDriverName = "pgx-rebind"

func init() {
	sql.Register(pgxRebindDriverName, &rebindDriver{base: stdlib.GetDefaultDriver()})
}

type rebindDriver struct {
	base driver.Driver
}

func (d *rebindDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.base.Open(name)
	if err != nil {
		return nil, err
	}
	return &rebindConn{Conn: conn}, nil
}

type rebindConn struct {
	driver.Conn
}

func (c *rebindConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.Conn.Prepare(Rebind("postgres", query))
	if err != nil {
		return nil, err
	}
	return &rebindStmt{Stmt: stmt}, nil
}

func (c *rebindConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if preparer, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := preparer.PrepareContext(ctx, Rebind("postgres", query))
		if err != nil {
			return nil, err
		}
		return &rebindStmt{Stmt: stmt}, nil
	}
	return c.Prepare(query)
}

func (c *rebindConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if beginner, ok := c.Conn.(driver.ConnBeginTx); ok {
		return beginner.BeginTx(ctx, opts)
	}
	return c.Conn.Begin()
}

func (c *rebindConn) Ping(ctx context.Context) error {
	if pinger, ok := c.Conn.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return nil
}

func (c *rebindConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if execer, ok := c.Conn.(driver.ExecerContext); ok {
		return execer.ExecContext(ctx, Rebind("postgres", query), args)
	}
	return nil, driver.ErrSkip
}

func (c *rebindConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if queryer, ok := c.Conn.(driver.QueryerContext); ok {
		return queryer.QueryContext(ctx, Rebind("postgres", query), args)
	}
	return nil, driver.ErrSkip
}

func (c *rebindConn) CheckNamedValue(value *driver.NamedValue) error {
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return nil
}

func (c *rebindConn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.Conn.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}
	return nil
}

type rebindStmt struct {
	driver.Stmt
}

func (s *rebindStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if execer, ok := s.Stmt.(driver.StmtExecContext); ok {
		return execer.ExecContext(ctx, args)
	}
	return nil, driver.ErrSkip
}

func (s *rebindStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if queryer, ok := s.Stmt.(driver.StmtQueryContext); ok {
		return queryer.QueryContext(ctx, args)
	}
	return nil, driver.ErrSkip
}

func (s *rebindStmt) ColumnConverter(index int) driver.ValueConverter {
	if converter, ok := s.Stmt.(driver.ColumnConverter); ok {
		return converter.ColumnConverter(index)
	}
	return driver.DefaultParameterConverter
}
