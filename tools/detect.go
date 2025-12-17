//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	switch os.Args[1] {
	case "proto":
		detectProto()
	case "wire":
		detectWire()
	case "services":
		detectServices()
	}
}

func detectProto() {
	var files []string
	filepath.Walk("api/proto", func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".proto") {
			rel, _ := filepath.Rel("api/proto", path)
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	fmt.Println(strings.Join(files, " "))
}

func detectWire() {
	var pkgs []string
	filepath.Walk("internal", func(path string, info os.FileInfo, err error) error {
		if info.Name() == "wire.go" {
			pkgs = append(pkgs, "./"+filepath.ToSlash(filepath.Dir(path)))
		}
		return nil
	})
	fmt.Println(strings.Join(pkgs, " "))
}

func detectServices() {
	entries, _ := os.ReadDir("cmd")
	var services []string
	for _, e := range entries {
		if e.IsDir() {
			services = append(services, e.Name())
		}
	}
	fmt.Println(strings.Join(services, " "))
}
