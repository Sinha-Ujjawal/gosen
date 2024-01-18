package main

import (
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// dirPath := "./docs.gl"
	dbName := "docs.gl.db"
	// fileContents, errs := FileContentsFromDirectory(dirPath)
	// for _, err := range errs {
	// 	if err != nil {
	// 		log.Fatalf("%s", err)
	// 	}
	// }
	// termFrequenciesIndex := MkTermFrequenciesIndexFromFileContents(fileContents)
	// // for filePath, tf := range termFrequenciesIndex.tfs {
	// // 	fmt.Printf("File %s has %d terms\n", filePath, len(tf))
	// // }
	// err := termFrequenciesIndex.DumpToSQLite3(dbName)
	// if err != nil {
	// 	log.Fatalf("%s", err)
	// }
	termFrequenciesIndex, err := LoadTermFrequenciesIndexFromSQLite3(dbName)
	if err != nil {
		log.Fatalf("%s", err)
	}
	for filePath, tf := range termFrequenciesIndex.tfs {
		fmt.Printf("File %s has %d terms\n", filePath, len(tf))
	}
	_ = termFrequenciesIndex
}
