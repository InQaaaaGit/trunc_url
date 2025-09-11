package buildinfo_test

import (
	"fmt"

	"github.com/InQaaaaGit/trunc_url.git/internal/buildinfo"
)

// ExampleDefaultInfo демонстрирует создание информации о сборке по умолчанию
func ExampleDefaultInfo() {
	info := buildinfo.DefaultInfo()
	fmt.Printf("Version: %s\n", info.Version)
	fmt.Printf("Date: %s\n", info.Date)
	fmt.Printf("Commit: %s\n", info.Commit)

	// Output:
	// Version: N/A
	// Date: N/A
	// Commit: N/A
}

// ExampleNewInfo демонстрирует создание информации о сборке с заданными параметрами
func ExampleNewInfo() {
	info := buildinfo.NewInfo("v1.0.0", "2024-01-01", "abc123")
	fmt.Printf("Version: %s\n", info.Version)
	fmt.Printf("Date: %s\n", info.Date)
	fmt.Printf("Commit: %s\n", info.Commit)

	// Output:
	// Version: v1.0.0
	// Date: 2024-01-01
	// Commit: abc123
}

// ExampleInfo_String демонстрирует получение строкового представления информации о сборке
func ExampleInfo_String() {
	info := buildinfo.NewInfo("v1.0.0", "2024-01-01", "abc123")
	fmt.Println(info.String())

	// Output:
	// Version: v1.0.0, Date: 2024-01-01, Commit: abc123
}
