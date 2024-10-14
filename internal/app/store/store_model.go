package store

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
)

type Store struct {
	dbName string
	DB     *sql.DB
}

func NewStore() *Store {
	dbDir := "./internal/app/store"

	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		err = os.Mkdir(dbDir, 0755)
		if err != nil {
			log.Fatal("ошибка при создании директории db: %w", err)
		}
	}

	dbFile := filepath.Join(dbDir, "scheduler.db")

	return &Store{
		dbName: dbFile,
	}
}
