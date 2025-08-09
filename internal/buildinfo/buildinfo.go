// Package buildinfo предоставляет функциональность для управления информацией о сборке приложения.
// Информация о сборке включает версию, дату сборки и commit hash.
package buildinfo

import "fmt"

// Info содержит информацию о сборке приложения
type Info struct {
	Version string
	Date    string
	Commit  string
}

// DefaultInfo возвращает информацию о сборке по умолчанию
func DefaultInfo() *Info {
	return &Info{
		Version: "N/A",
		Date:    "N/A",
		Commit:  "N/A",
	}
}

// NewInfo создает новую структуру с информацией о сборке
func NewInfo(version, date, commit string) *Info {
	return &Info{
		Version: version,
		Date:    date,
		Commit:  commit,
	}
}

// Print выводит информацию о сборке в консоль
func (info *Info) Print() {
	fmt.Printf("Build version: %s\n", info.Version)
	fmt.Printf("Build date: %s\n", info.Date)
	fmt.Printf("Build commit: %s\n", info.Commit)
}

// String возвращает строковое представление информации о сборке
func (info *Info) String() string {
	return fmt.Sprintf("Version: %s, Date: %s, Commit: %s", info.Version, info.Date, info.Commit)
}
