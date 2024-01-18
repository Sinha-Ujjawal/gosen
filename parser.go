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

// FileContent struct for storing the file path of the file as well as the content
type FileContent struct {
	filePath string
	content  string
}

func FileContentFromFilePath(filePath string) (*FileContent, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("FileContentFromFilePath: failed reading the filePath %s: %w", filePath, err)
	}
	reader := bufio.NewReader(fp)
	handler := &TextHandler{}
	parser := saxlike.NewParser(reader, handler)
	err = parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("FileContentFromFilePath: failed parsing the file %s using saxlike: %w", filePath, err)
	}
	return &FileContent{filePath, handler.textDataSB.String()}, nil
}

func FileContentsFromDirectory(dirPath string) ([]FileContent, []error) {
	files, err := filepath.Glob(dirPath + "/*/*.xhtml")
	if err != nil {
		return nil, []error{fmt.Errorf("FileContentsFromDirectory: failed reading .xhtml files from the directory %s: %w", dirPath, err)}
	}
	var fileContents []FileContent
	var errs []error
	lock := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(files))
	for _, filePath := range files {
		filePath, _ := filepath.Abs(filePath)
		go func(filePath string) {
			defer wg.Done()
			fileContent, err := FileContentFromFilePath(filePath)
			lock.Lock()
			defer lock.Unlock()
			if err != nil {
				errs = append(errs, err)
			} else {
				fileContents = append(fileContents, *fileContent)
			}
		}(filePath)
	}
	wg.Wait()
	return fileContents, errs
}
