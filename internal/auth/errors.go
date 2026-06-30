package auth

import "errors"

var (
	ErrInvalidAPIKey  = errors.New("API Key không được để trống")
	ErrAPIKeyExists   = errors.New("API Key đã tồn tại")
	ErrAPIKeyNotFound = errors.New("API Key không tồn tại")
	ErrDeleteAdminKey = errors.New("không thể xóa khóa quản trị")
)
