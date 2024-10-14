package store

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func (s *Store) Open() error {
	install := s.CheckExistDB()

	db, err := sql.Open("sqlite", s.dbName)

	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	s.DB = db

	if err := s.CreatingTable(install); err != nil {
		log.Println("таблица уже создана")
	}

	return nil
}

func (s *Store) Close() {
	s.DB.Close()
}

func (s *Store) CheckExistDB() bool {
	var install bool

	if _, err := os.Stat(s.dbName); err == nil {
		install = true
	}

	return install
}
