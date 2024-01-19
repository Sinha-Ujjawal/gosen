package main

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"gosen/saxlike"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SAX Like XML Handler for capturing all strings from the file content
type TextHandler struct {
	saxlike.VoidHandler
	textDataSB strings.Builder
}

// Handler function to append string from the file to intermediary results
func (h *TextHandler) CharData(c xml.CharData) {
	h.textDataSB.Write(c)
	h.textDataSB.WriteString(" ")
}

func FileContentFromFilePath(filePath string) (string, error) {
	parts := strings.Split(filePath, ".")
	ext := parts[len(parts)-1]
	switch ext {
	case "xhtml", "html", "xml", "svg":
		fp, err := os.Open(filePath)
		if err != nil {
			return "", fmt.Errorf("FileContentFromFilePath: failed reading the filePath %s: %w", filePath, err)
		}
		reader := bufio.NewReader(fp)
		handler := &TextHandler{}
		parser := saxlike.NewParser(reader, handler)
		err = parser.Parse()
		if err != nil {
			return "", fmt.Errorf("FileContentFromFilePath: failed parsing the file %s using saxlike: %w", filePath, err)
		}
		return handler.textDataSB.String(), nil
	default:
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("FileContentFromFilePath: failed reading the filePath %s: %w", filePath, err)
		}
		return string(bytes), nil
	}
}

func ListFiles(directory string) ([]string, error) {
	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func FileContentsFromDirectory(dirPath string) (map[string]string, []error) {
	files, err := ListFiles(dirPath)
	if err != nil {
		return nil, []error{fmt.Errorf("FileContentsFromDirectory: failed reading files from the directory %s: %w", dirPath, err)}
	}
	fileContents := map[string]string{}
	var errs []error
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, filePath := range files {
		filePath, _ := filepath.Abs(filePath)
		if fi, _ := os.Stat(filePath); fi.Mode().IsRegular() {
			wg.Add(1)
			go func(filePath string) {
				defer wg.Done()
				fileContent, err := FileContentFromFilePath(filePath)
				lock.Lock()
				defer lock.Unlock()
				if err != nil {
					errs = append(errs, err)
				} else {
					fileContents[filePath] = fileContent
				}
			}(filePath)
		}
	}
	wg.Wait()
	return fileContents, errs
}
