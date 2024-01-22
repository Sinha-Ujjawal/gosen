package fileContents

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"gosen/saxlike"
	"gosen/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ledongthuc/pdf"
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
	defer fp.Close()
	if err != nil {
		return "", fmt.Errorf("readXML: failed reading the filePath %s: %w", filePath, err)
	}
	reader := bufio.NewReader(fp)
	handler := &textHandler{}
	parser := saxlike.NewParser(reader, handler)
	err = parser.Parse()
	if err != nil {
		return "", fmt.Errorf("readXML: failed parsing the file %s using saxlike: %w", filePath, err)
	}
	return handler.textDataSB.String(), nil
}

func readPDF(path string) (string, bool) {
	defer func() {
		if err := recover(); err != nil {
			slog.Errorf("panic occurred: %s", err)
		}
	}()
	f, r, err := pdf.Open(path)
	// remember close file
	defer f.Close()
	if err != nil {
		// TODO: handle malformed pdfs
		slog.Errorf("readPDF: failed to open the file `%s`: %s!; returning with empty string", path, err)
		return "", false
	}
	rio, err := r.GetPlainText()
	if err != nil {
		// TODO: handle malformed pdfs
		slog.Errorf("readPDF: failed to get the plantext for the file `%s`: %s!; returning with empty string", path, err)
		return "", false
	}
	reader := bufio.NewReader(rio)
	sb := strings.Builder{}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String(), true
}

func readText(filePath string) (string, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("readText: failed reading the filePath %s: %w", filePath, err)
	}
	return string(bytes), nil
}

func FromFilePath(filePath string) (string, error) {
	parts := strings.Split(filePath, ".")
	ext := parts[len(parts)-1]
	switch ext {
	case "xhtml", "html", "xml", "svg":
		return readXML(filePath)
	case "pdf":
		text, noError := readPDF(filePath)
		if !noError {
			slog.Infof("FromFilePath: Unable to extract text from pdf `%s`, try reading it as plain text", filePath)
			return readText(filePath)
		}
		return text, nil
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

func FromDirectory[T any](dirPath string, processContent func(string) T) (map[string]T, []error) {
	files, err := listFiles(dirPath)
	if err != nil {
		return nil, []error{fmt.Errorf("FromDirectory: failed reading files from the directory %s: %w", dirPath, err)}
	}
	fileContents := map[string]T{}
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
					fileContents[filePath] = processContent(fileContent)
				}
			}(filePath)
		}
	}
	wg.Wait()
	return fileContents, errs
}
