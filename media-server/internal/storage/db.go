// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package storage

import (
	"context"
	"database/sql"
	"fmt"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.attachUserPrivateKeyStmt, err = db.PrepareContext(ctx, attachUserPrivateKey); err != nil {
		return nil, fmt.Errorf("error preparing query AttachUserPrivateKey: %w", err)
	}
	if q.attachUserRefreshTokenStmt, err = db.PrepareContext(ctx, attachUserRefreshToken); err != nil {
		return nil, fmt.Errorf("error preparing query AttachUserRefreshToken: %w", err)
	}
	if q.delRefreshTokenStmt, err = db.PrepareContext(ctx, delRefreshToken); err != nil {
		return nil, fmt.Errorf("error preparing query DelRefreshToken: %w", err)
	}
	if q.detachUserPrivateKeyStmt, err = db.PrepareContext(ctx, detachUserPrivateKey); err != nil {
		return nil, fmt.Errorf("error preparing query DetachUserPrivateKey: %w", err)
	}
	if q.detachUserRefreshTokenStmt, err = db.PrepareContext(ctx, detachUserRefreshToken); err != nil {
		return nil, fmt.Errorf("error preparing query DetachUserRefreshToken: %w", err)
	}
	if q.getPrivateKeyStmt, err = db.PrepareContext(ctx, getPrivateKey); err != nil {
		return nil, fmt.Errorf("error preparing query GetPrivateKey: %w", err)
	}
	if q.getPrivateKeyWithUserStmt, err = db.PrepareContext(ctx, getPrivateKeyWithUser); err != nil {
		return nil, fmt.Errorf("error preparing query GetPrivateKeyWithUser: %w", err)
	}
	if q.getUserStmt, err = db.PrepareContext(ctx, getUser); err != nil {
		return nil, fmt.Errorf("error preparing query GetUser: %w", err)
	}
	if q.getUserByUsernameStmt, err = db.PrepareContext(ctx, getUserByUsername); err != nil {
		return nil, fmt.Errorf("error preparing query GetUserByUsername: %w", err)
	}
	if q.getUserPrivateKeyStmt, err = db.PrepareContext(ctx, getUserPrivateKey); err != nil {
		return nil, fmt.Errorf("error preparing query GetUserPrivateKey: %w", err)
	}
	if q.newPrivateKeyStmt, err = db.PrepareContext(ctx, newPrivateKey); err != nil {
		return nil, fmt.Errorf("error preparing query NewPrivateKey: %w", err)
	}
	if q.newRefreshTokenStmt, err = db.PrepareContext(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("error preparing query NewRefreshToken: %w", err)
	}
	if q.newUserStmt, err = db.PrepareContext(ctx, newUser); err != nil {
		return nil, fmt.Errorf("error preparing query NewUser: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.attachUserPrivateKeyStmt != nil {
		if cerr := q.attachUserPrivateKeyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing attachUserPrivateKeyStmt: %w", cerr)
		}
	}
	if q.attachUserRefreshTokenStmt != nil {
		if cerr := q.attachUserRefreshTokenStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing attachUserRefreshTokenStmt: %w", cerr)
		}
	}
	if q.delRefreshTokenStmt != nil {
		if cerr := q.delRefreshTokenStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing delRefreshTokenStmt: %w", cerr)
		}
	}
	if q.detachUserPrivateKeyStmt != nil {
		if cerr := q.detachUserPrivateKeyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing detachUserPrivateKeyStmt: %w", cerr)
		}
	}
	if q.detachUserRefreshTokenStmt != nil {
		if cerr := q.detachUserRefreshTokenStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing detachUserRefreshTokenStmt: %w", cerr)
		}
	}
	if q.getPrivateKeyStmt != nil {
		if cerr := q.getPrivateKeyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPrivateKeyStmt: %w", cerr)
		}
	}
	if q.getPrivateKeyWithUserStmt != nil {
		if cerr := q.getPrivateKeyWithUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPrivateKeyWithUserStmt: %w", cerr)
		}
	}
	if q.getUserStmt != nil {
		if cerr := q.getUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getUserStmt: %w", cerr)
		}
	}
	if q.getUserByUsernameStmt != nil {
		if cerr := q.getUserByUsernameStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getUserByUsernameStmt: %w", cerr)
		}
	}
	if q.getUserPrivateKeyStmt != nil {
		if cerr := q.getUserPrivateKeyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getUserPrivateKeyStmt: %w", cerr)
		}
	}
	if q.newPrivateKeyStmt != nil {
		if cerr := q.newPrivateKeyStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing newPrivateKeyStmt: %w", cerr)
		}
	}
	if q.newRefreshTokenStmt != nil {
		if cerr := q.newRefreshTokenStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing newRefreshTokenStmt: %w", cerr)
		}
	}
	if q.newUserStmt != nil {
		if cerr := q.newUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing newUserStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                         DBTX
	tx                         *sql.Tx
	attachUserPrivateKeyStmt   *sql.Stmt
	attachUserRefreshTokenStmt *sql.Stmt
	delRefreshTokenStmt        *sql.Stmt
	detachUserPrivateKeyStmt   *sql.Stmt
	detachUserRefreshTokenStmt *sql.Stmt
	getPrivateKeyStmt          *sql.Stmt
	getPrivateKeyWithUserStmt  *sql.Stmt
	getUserStmt                *sql.Stmt
	getUserByUsernameStmt      *sql.Stmt
	getUserPrivateKeyStmt      *sql.Stmt
	newPrivateKeyStmt          *sql.Stmt
	newRefreshTokenStmt        *sql.Stmt
	newUserStmt                *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db:                         tx,
		tx:                         tx,
		attachUserPrivateKeyStmt:   q.attachUserPrivateKeyStmt,
		attachUserRefreshTokenStmt: q.attachUserRefreshTokenStmt,
		delRefreshTokenStmt:        q.delRefreshTokenStmt,
		detachUserPrivateKeyStmt:   q.detachUserPrivateKeyStmt,
		detachUserRefreshTokenStmt: q.detachUserRefreshTokenStmt,
		getPrivateKeyStmt:          q.getPrivateKeyStmt,
		getPrivateKeyWithUserStmt:  q.getPrivateKeyWithUserStmt,
		getUserStmt:                q.getUserStmt,
		getUserByUsernameStmt:      q.getUserByUsernameStmt,
		getUserPrivateKeyStmt:      q.getUserPrivateKeyStmt,
		newPrivateKeyStmt:          q.newPrivateKeyStmt,
		newRefreshTokenStmt:        q.newRefreshTokenStmt,
		newUserStmt:                q.newUserStmt,
	}
}
