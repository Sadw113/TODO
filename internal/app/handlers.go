package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"todo/internal/models"
)

const layoutDate = "20060102"

func (s *APIServer) NextDateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nowParam := r.FormValue("now")
		dateParam := r.FormValue("date")
		repeatParam := r.FormValue("repeat")

		if nowParam == "" || dateParam == "" || repeatParam == "" {
			http.Error(w, "не заданы параметры", http.StatusBadRequest)
			return
		}
		now, err := time.Parse(layoutDate, nowParam)
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
		task.Date = time.Now().Format(layoutDate)
	}

	taskDate, err := time.Parse(layoutDate, task.Date)
	if err != nil {
		http.Error(w, `{"error":"неправильный формат даты"}`, http.StatusBadRequest)
		return
	}

	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	taskDateTruncated := taskDate.Truncate(24 * time.Hour)
	if taskDateTruncated.Equal(today) {
		task.Date = today.Format(layoutDate)
	} else if taskDate.Before(now) {
		if task.Repeat == "" {
			task.Date = now.Format(layoutDate)
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

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "Не указан идентификатор"}`, http.StatusBadRequest)
		return
	}

	_, err := s.store.GetTaskByID(id, &task)

	if err != nil {
		http.Error(w, `{"error": "Задача не найдена"}`, http.StatusBadRequest)
		return
	}

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
		task.Date = time.Now().Format(layoutDate)
	}

	taskDate, err := time.Parse(layoutDate, task.Date)
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
			task.Date = now.Format(layoutDate)
		}
	}

	err = s.store.UpdateTask(task)
	if err != nil {
		http.Error(w, `{"error":"Ошибка обновления в базе данных"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{}`))
}

func (s *APIServer) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	if taskID == "" {
		http.Error(w, `{"error":"Не указан идентификатор задачи"}`, http.StatusBadRequest)
		return
	}

	err := s.store.DeleteTask(taskID)
	if err != nil {
		http.Error(w, `{"error":"Ошибка при удалении задачи"}`, http.StatusInternalServerError)
		return

	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{}`))
}

func (s *APIServer) DoneTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var task models.Task

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"Не указан идентификатор"}`, http.StatusBadRequest)
			return
		}

		_, err := s.store.GetTaskByID(id, &task)
		if err != nil {
			http.Error(w, `{"error": "Задача не найдена"}`, http.StatusBadRequest)
			return
		}

		if task.Repeat == "" {
			err := s.store.DeleteTask(task.Id)
			if err != nil {
				http.Error(w, `{"error":"Ошибка при удалении задачи"}`, http.StatusInternalServerError)
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

			err = s.store.SetDate(task, nextDate)
			if err != nil {
				http.Error(w, `{"error":"Ошибка обновления в базе данных"}`, http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}
}

func NextDate(now time.Time, date string, repeat string) (string, error) {
	dateTime, err := time.Parse(layoutDate, date)

	if err != nil {
		return "", fmt.Errorf("ошибка при разборе даты: %v", err)
	}

	if repeat == "" {
		return "", fmt.Errorf("правило повторения не задано")
	}

	rep_slice := strings.Split(repeat, " ")

	if len(rep_slice) < 1 {
		return "", fmt.Errorf("неправильный формат правила повторения")
	}

	switch rep_slice[0] {
	case "d":
		if len(rep_slice) < 2 {
			return "", fmt.Errorf("не указано количество дней")
		}
		interval, err := strconv.Atoi(rep_slice[1])
		if err != nil || interval < 1 || interval > 400 {
			return "", fmt.Errorf("некорретный интервал для повторения")
		}

		if dateTime.After(now) {
			dateTime = dateTime.AddDate(0, 0, interval)
		}
		for dateTime.Before(now) {
			dateTime = dateTime.AddDate(0, 0, interval)
		}

		return dateTime.Format(layoutDate), nil

	case "y":
		if dateTime.After(now) {
			dateTime = dateTime.AddDate(1, 0, 0)
		}

		for dateTime.Before(now) {
			dateTime = dateTime.AddDate(1, 0, 0)
		}

		nextDateValue := dateTime.Format(layoutDate)

		return nextDateValue, nil

	default:
		return "", fmt.Errorf("неподдерживаемый формат правила повторения")
	}
}
