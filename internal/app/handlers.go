package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"todo/models"
)

func (s *APIServer) NextDateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nowParam := r.FormValue("now")
		dateParam := r.FormValue("date")
		repeatParam := r.FormValue("repeat")

		if nowParam == "" || dateParam == "" || repeatParam == "" {
			http.Error(w, "не заданы параметры", http.StatusBadRequest)
			return
		}
		now, err := time.Parse("20060102", nowParam)
		if err != nil {
			http.Error(w, "неверный формат времени", http.StatusBadRequest)
			return
		}
		nextDate, err := NextDate(now, dateParam, repeatParam)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		io.WriteString(w, nextDate)
	}
}

func (s *APIServer) ApiTaskMethods() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			s.AddTask(w, r)
		case http.MethodGet:
			s.GetTask(w, r)
		case http.MethodPut:
			s.UpdateTask(w, r)
		case http.MethodDelete:
			s.DeleteTask(w, r)
		default:
			http.Error(w, `{"error":"метод не поддерживается"}`, http.StatusMethodNotAllowed)
		}
	}
}

func (s *APIServer) AddTask(w http.ResponseWriter, r *http.Request) {
	var task *models.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, `{"error":"ошибка десериализации JSON"}`, http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, `{"error":"не указан заголовок задачи"}`, http.StatusBadRequest)
		return
	}

	if task.Date == "" {
		task.Date = time.Now().Format("20060102")
	}

	taskDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		http.Error(w, `{"error":"неправильный формат даты"}`, http.StatusBadRequest)
		return
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	taskDateTruncated := taskDate.Truncate(24 * time.Hour)
	if taskDateTruncated.Equal(today) {
		task.Date = today.Format("20060102")
	} else if taskDate.Before(now) {
		if task.Repeat == "" {
			task.Date = now.Format("20060102")
		} else {
			nextDate, err := NextDate(now, task.Date, task.Repeat)
			if err != nil {
				http.Error(w, `{"error":"ошибка при вычислении следующей даты"}`, http.StatusBadRequest)
				return
			}
			task.Date = nextDate
		}
	}

	resInsert, err := s.store.InsertTask(task)
	if err != nil {
		http.Error(w, `{"error":"ошибка при добавлении задачи"}`, http.StatusInternalServerError)
		return
	}

	id, err := s.store.GetLastInsertId(resInsert)
	if err != nil {
		http.Error(w, `{"error":"ошибка при получении последнего добавленного id"}`, http.StatusInternalServerError)
		return
	}

	res := map[string]any{
		"id": fmt.Sprintf("%d", id),
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	errEncode := json.NewEncoder(w).Encode(res)
	if errEncode != nil {
		http.Error(w, `{"error":"Не указан заголовок задачи"}`, http.StatusInternalServerError)
		return
	}
}

func (s *APIServer) GetTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tasks []models.Task

		err := s.store.GetTasks(&tasks)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		if tasks == nil {
			tasks = []models.Task{}
		}

		res := map[string]interface{}{
			"tasks": tasks,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		errEncode := json.NewEncoder(w).Encode(res)
		if errEncode != nil {
			http.Error(w, `{"error":"Ошибка вывода задач"}`, http.StatusInternalServerError)
			return
		}
	}
}

func (s *APIServer) GetTask(w http.ResponseWriter, r *http.Request) {
	var task models.Task
	var needRepeat bool

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	selectedID, _, err := s.store.GetTask(id, &task, needRepeat)
	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusBadRequest)
		return
	}

	task.Id = fmt.Sprint(selectedID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *APIServer) UpdateTask(w http.ResponseWriter, r *http.Request) {
	var task models.Task

	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, `{"error":"Ошибка десериализации JSON"}`, http.StatusBadRequest)
		return
	}

	if task.Id == "" {
		http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		http.Error(w, `{"error":"Не указан заголовок"}`, http.StatusBadRequest)
		return
	}

	if task.Date == "" {
		task.Date = time.Now().Format("20060102")
	}

	taskDate, err := time.Parse("20060102", task.Date)
	if err != nil {
		http.Error(w, `{"error":"Неверная дата"}`, http.StatusBadRequest)
		return
	}

	now := time.Now()
	if taskDate.Before(now) {
		if task.Repeat != "" {
			nextDate, err := NextDate(now, task.Date, task.Repeat)
			if err != nil {
				http.Error(w, `{"error":"Ошибка при вычислении следующей даты"}`, http.StatusBadRequest)
				return
			}
			task.Date = nextDate
		} else {
			task.Date = now.Format("20060102")
		}
	}

	rowsAffected, errStr := s.store.UpdateTask(task, false, "")
	if errStr != "" {
		if errStr == "Ошибка обновления в базе данных" {
			http.Error(w, `{"error":"Ошибка обновления в базе данных"}`, http.StatusInternalServerError)
			return
		}
		if errStr == "Ошибка при получении результата обновления" {
			http.Error(w, `{"error":"Ошибка при задачи для обновления"}`, http.StatusInternalServerError)
			return
		}
	}
	if rowsAffected == 0 {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func (s *APIServer) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	rowsAffected, errStr := s.store.DeleteTask(taskID)
	if errStr != "" {
		if errStr == "Ошибка при удалении задачи" {
			http.Error(w, `{"error":"Ошибка при удалении задачи"}`, http.StatusInternalServerError)
			return
		}
		if errStr == "Ошибка при поиске задачи для удаления" {
			http.Error(w, `{"error":"Ошибка получения информации об удалении"}`, http.StatusInternalServerError)
			return
		}
	}

	if rowsAffected == 0 {
		http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{}`))
}

func (s *APIServer) DoneTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var task models.Task
		needUpdateOnlyDate := true

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
			return
		}

		selectedID, repeat, err := s.store.GetTask(id, &task, needUpdateOnlyDate)
		if err != nil {
			http.Error(w, `{"error": "Задача не найдена"}`, http.StatusBadRequest)
			return
		}

		task.Id = fmt.Sprint(selectedID)
		task.Repeat = repeat

		if task.Repeat == "" {
			rowsAffected, errStr := s.store.DeleteTask(task.Id)
			if errStr != "" {
				if errStr == "Ошибка при удалении задачи" {
					http.Error(w, `{"error":"Ошибка при удалении задачи"}`, http.StatusInternalServerError)
					return
				}
				if errStr == "Ошибка при поиске задачи для удаления" {
					http.Error(w, `{"error":"Ошибка получения информации об удалении"}`, http.StatusInternalServerError)
					return
				}
			}

			if rowsAffected == 0 {
				http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
				return
			}
		}

		if task.Repeat != "" {
			now := time.Now()
			nextDate, err := NextDate(now, task.Date, task.Repeat)
			if err != nil {
				http.Error(w, `{"error":"Ошибка при вычислении следующей даты"}`, http.StatusBadRequest)
				return
			}

			rowsAffected, errStr := s.store.UpdateTask(task, true, nextDate)
			if errStr != "" {
				if errStr == "Ошибка обновления в базе данных" {
					http.Error(w, `{"error":"Ошибка обновления в базе данных"}`, http.StatusInternalServerError)
					return
				}
				if errStr == "Ошибка при получении результата обновления" {
					http.Error(w, `{"error":"Ошибка при задачи для обновления"}`, http.StatusInternalServerError)
					return
				}
			}
			if rowsAffected == 0 {
				http.Error(w, `{"error":"Задача не найдена"}`, http.StatusNotFound)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}
}
