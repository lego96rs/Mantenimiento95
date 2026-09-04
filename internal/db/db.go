package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"
)

type DB struct {
	Write *sql.DB
	Read  *sql.DB
}

func Open(path string) (*DB, error) {
	write, err := open(path, true)
	if err != nil {
		return nil, fmt.Errorf("open write pool: %w", err)
	}

	read, err := open(path, false)
	if err != nil {
		write.Close()
		return nil, fmt.Errorf("open read pool: %w", err)
	}

	database := &DB{
		Write: write,
		Read:  read,
	}

	if err := database.Ping(context.Background()); err != nil {
		database.Close()
		return nil, err
	}

	return database, nil
}

func open(path string, writer bool) (*sql.DB, error) {
	query := url.Values{}
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "journal_mode(WAL)")
	query.Add("_pragma", "synchronous(NORMAL)")
	query.Add("_pragma", "foreign_keys(1)")
	if writer {
		query.Set("_txlock", "immediate")
	}

	pool, err := sql.Open("sqlite", "file:"+path+"?"+query.Encode())
	if err != nil {
		return nil, err
	}

	if writer {
		pool.SetMaxOpenConns(1)
	} else {
		pool.SetMaxOpenConns(8)
	}

	return pool, nil
}

func (d *DB) Ping(ctx context.Context) error {
	if err := d.Write.PingContext(ctx); err != nil {
		return fmt.Errorf("ping write pool: %w", err)
	}
	if err := d.Read.PingContext(ctx); err != nil {
		return fmt.Errorf("ping read pool: %w", err)
	}
	return nil
}

func (d *DB) Close() error {
	writeErr := d.Write.Close()
	readErr := d.Read.Close()
	if writeErr != nil {
		return writeErr
	}
	return readErr
}
