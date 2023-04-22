package webapp

import (
	"database/sql"
	_ "github.com/lib/pq"
)

var GlobalPostgresDB *sql.DB // MySQL Database

func NewPostgresDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn+"?sslmode=disable")
	if err != nil {
		return nil, err
	}
	return db, db.Ping()
}
