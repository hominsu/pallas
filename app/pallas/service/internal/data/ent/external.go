package ent

import (
	"context"
	"database/sql"
	"fmt"
)

// ExecContext allows calling the underlying ExecContext method of the driver if it is supported by it.
// See, database/sql#DB.ExecContext for more information.
func (c *config) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ex, ok := c.driver.(interface {
		ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.ExecContext is not supported")
	}
	return ex.ExecContext(ctx, query, args...)
}

// QueryContext allows calling the underlying QueryContext method of the driver if it is supported by it.
// See, database/sql#DB.QueryContext for more information.
func (c *config) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	q, ok := c.driver.(interface {
		QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.QueryContext is not supported")
	}
	return q.QueryContext(ctx, query, args...)
}
