package store

import (
	"database/sql"
	"fmt"
	"log"
	"todo/models"
)

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

// Второй параметр needRepeat нужен в случае, когда требуется отметить задачу выполненной.
// Если он равен false, то вернется идентификатор задачи и ошибка
// Если он равен true, то вернется ещё и повторение задачи
func (s *Store) GetTask(id string, task *models.Task, needRepeat bool) (int64, string, error) {
	var selectedID int64
	var repeat string

	if needRepeat {
		query := "SELECT * FROM scheduler WHERE id = ?"
		row := s.DB.QueryRow(query, id)

		err := row.Scan(&selectedID, &task.Date, &task.Title, &task.Comment, &repeat)
		if err != nil {
			return 0, "", err
		}

		return selectedID, repeat, nil
	}

	query := "SELECT * FROM scheduler WHERE id = ?"
	row := s.DB.QueryRow(query, id)

	err := row.Scan(&selectedID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	if err != nil {
		return 0, "", err
	}

	return selectedID, "", nil
}

// Второй параметр needUpdateOnlyDate нужен, когда требуется отметить задачу выполненной.
// Если он равен true, то тогда обновится только значение даты в таблице, в таком случае нужен третий параметр со следующей датой выполнения
// Если он равен false, то выполнится классическое обновление
func (s *Store) UpdateTask(task models.Task, needUpdateOnlyDate bool, nextDate string) (int64, string) {
	if needUpdateOnlyDate {
		query := "UPDATE scheduler SET date = ? WHERE id = ?"

		res, err := s.DB.Exec(query, nextDate, task.Id)
		if err != nil {
			return 0, "Ошибка обновления в базе данных"
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return 0, "Ошибка при получении результата обновления"
		}

		return rowsAffected, ""
	}

	query := "UPDATE scheduler SET date = ?, title = ?, comment = ?, repeat = ? WHERE id = ?"

	res, err := s.DB.Exec(query, task.Date, task.Title, task.Comment, task.Repeat, task.Id)
	if err != nil {
		return 0, "Ошибка обновления в базе данных"
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, "Ошибка при получении результата обновления"
	}

	return rowsAffected, ""
}

func (s *Store) DeleteTask(id string) (int64, string) {
	query := "DELETE FROM scheduler where id = ?"

	res, err := s.DB.Exec(query, id)
	if err != nil {
		return 0, "Ошибка при удалении задачи"
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, "Ошибка при поиске задачи для удаления"
	}

	return rowsAffected, ""
}
