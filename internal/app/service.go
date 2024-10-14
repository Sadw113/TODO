package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {
	dateTime, err := time.Parse("20060102", date)

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

		return dateTime.Format("20060102"), nil

	case "y":
		if dateTime.After(now) {
			dateTime = dateTime.AddDate(1, 0, 0)
		}

		for dateTime.Before(now) {
			dateTime = dateTime.AddDate(1, 0, 0)
		}

		nextDateValue := dateTime.Format("20060102")

		return nextDateValue, nil

	default:
		return "", fmt.Errorf("неподдерживаемый формат правила повторения")
	}
}
