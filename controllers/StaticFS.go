package controllers

import (
	"embed"
	"log"
)

var staticFS embed.FS

func InitStaticFS(fs embed.FS) {
	staticFS = fs
}

func GetStaticFileContent(relativePath string) ([]byte, error) {
	fullPath := "public/" + relativePath
	content, err := staticFS.ReadFile(fullPath)
	if err != nil {
		log.Printf("读取文件失败: %v\n", err)
		return nil, err
	}
	return content, nil
}
