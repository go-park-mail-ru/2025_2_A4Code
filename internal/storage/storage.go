package storage

import "errors"

// TODO: добавить ошибки при работе с данными в бд
var (
	ErrNotFound = errors.New("not found")
)
