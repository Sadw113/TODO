package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"todo/internal/models"

	_ "modernc.org/sqlite"
)

type Store struct {
	dbName string
	DB     *sql.DB
}

func NewStore() *Store {
	dbDir := "./store"

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

func (s *Store) CreatingTable(install bool) error {
	if !install {
		log.Fatal("нет файла .db")
	}

	query := `
		CREATE TABLE scheduler (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date TEXT,
			title TEXT NOT NULL,
			comment TEXT,
			repeat CHAR(128));

		CREATE INDEX repeat_duration ON scheduler (date);
	`

	_, err := s.DB.Exec(query)

	if err != nil {
		return err
	}

	return nil
}

func (s *Store) InsertTask(task *models.Task) (sql.Result, error) {
	query := "INSERT INTO scheduler (date, title, comment, repeat) VALUES (?, ?, ?, ?)"
	res, err := s.DB.Exec(query, task.Date, task.Title, task.Comment, task.Repeat)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Store) GetLastInsertId(res sql.Result) (int64, error) {
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Store) GetTasks(tasks *[]models.Task) error {
	query := "SELECT * FROM scheduler ORDER BY date LIMIT 30"
	rows, err := s.DB.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var task models.Task
		var id int64

		err := rows.Scan(&id, &task.Date, &task.Title, &task.Comment, &task.Repeat)
		if err != nil {
			return err
		}

		task.Id = fmt.Sprint(id)
		*tasks = append(*tasks, task)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Store) GetTaskByID(id string, task *models.Task) (*models.Task, error) {
	var selectedID int

	query := "SELECT * FROM scheduler WHERE id = ?"
	row := s.DB.QueryRow(query, id)

	err := row.Scan(&selectedID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	task.Id = strconv.Itoa(selectedID)
	if err != nil {
		return task, err
	}

	return task, nil
}

func (s *Store) UpdateTask(task models.Task) error {
	query := "UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?"

	res, err := s.DB.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.Id)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) SetDate(task models.Task, nextDate string) error {
	query := "UPDATE scheduler SET date = ? WHERE id = ?"

	_, err := s.DB.Exec(query, nextDate, task.Id)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) DeleteTask(id string) error {
	query := "DELETE FROM scheduler where id = ?"

	res, err := s.DB.Exec(query, id)
	if err != nil {
		return err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}
