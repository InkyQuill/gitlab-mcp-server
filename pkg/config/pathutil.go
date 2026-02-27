package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SafeJoinPath безопасно объединяет базовый путь и относительный путь
// Предотвращает directory traversal атаки
func SafeJoinPath(base, path string) (string, error) {
	// Очистить путь
	cleanPath := filepath.Clean(path)

	// Если путь абсолютный - отвергнуть (только для относительных путей)
	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("absolute paths not allowed: %s", cleanPath)
	}

	// Проверить на directory traversal
	if strings.Contains(cleanPath, "..") {
		return "", fmt.Errorf("directory traversal not allowed: %s", cleanPath)
	}

	// Объединить с базовым путём
	fullPath := filepath.Join(base, cleanPath)

	// Проверить что результат внутри базовой директории
	rel, err := filepath.Rel(base, fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes base directory: %s", path)
	}

	return fullPath, nil
}

// ValidatePath проверяет что путь безопасен для использования
func ValidatePath(path string) error {
	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("directory traversal not allowed: %s", cleanPath)
	}

	return nil
}

// SafeReadFile безопасно читает файл с защитой от path traversal
func SafeReadFile(baseDir, filename string) ([]byte, error) {
	fullPath, err := SafeJoinPath(baseDir, filename)
	if err != nil {
		return nil, err
	}

	// Дополнительная проверка что файл существует и это файл
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return nil, fmt.Errorf("expected a file, got directory: %s", filename)
	}

	return os.ReadFile(fullPath)
}

// SafeWriteFile безопасно записывает файл с защитой от path traversal
func SafeWriteFile(baseDir, filename string, data []byte, perm os.FileMode) error {
	fullPath, err := SafeJoinPath(baseDir, filename)
	if err != nil {
		return err
	}

	// Create parent directory if it doesn't exist
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	return os.WriteFile(fullPath, data, perm)
}
