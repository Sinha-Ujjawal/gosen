package fileContents

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

type textHandler struct {
	saxlike.VoidHandler
	textDataSB strings.Builder
}

func (h *textHandler) CharData(c xml.CharData) {
	h.textDataSB.Write(c)
	h.textDataSB.WriteString(" ")
}

func readXML(filePath string) (string, error) {
	fp, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("ReadXML: failed reading the filePath %s: %w", filePath, err)
	}
	reader := bufio.NewReader(fp)
	handler := &textHandler{}
	parser := saxlike.NewParser(reader, handler)
	err = parser.Parse()
	if err != nil {
		return "", fmt.Errorf("ReadXML: failed parsing the file %s using saxlike: %w", filePath, err)
	}
	return handler.textDataSB.String(), nil
}

func readText(filePath string) (string, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("ReadText: failed reading the filePath %s: %w", filePath, err)
	}
	return string(bytes), nil
}

func FromFilePath(filePath string) (string, error) {
	parts := strings.Split(filePath, ".")
	ext := parts[len(parts)-1]
	switch ext {
	case "xhtml", "html", "xml", "svg":
		return readXML(filePath)
	default:
		return readText(filePath)
	}
}

func listFiles(directory string) ([]string, error) {
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

func FromDirectory(dirPath string) (map[string]string, []error) {
	files, err := listFiles(dirPath)
	if err != nil {
		return nil, []error{fmt.Errorf("FromDirectory: failed reading files from the directory %s: %w", dirPath, err)}
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
				fileContent, err := FromFilePath(filePath)
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
