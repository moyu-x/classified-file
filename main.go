package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// PreMap 准备初始化数据
func PreMap() (map[string]int, map[string]int) {

	videoExt := [...]string{"mp4", "mkv", "mov", "webm", "avi", "wmv", "mpg",
		"flv", "3gp", "rmvb", "MP4", "MPG", "ass", "srt", "ssa", "MOV"}
	imageExt := [...]string{"jpg", "png", "gif", "webp", "jpeg", "JPG", "PNG",
		"bmp"}
	videoExtMap := make(map[string]int)
	imageExtMap := make(map[string]int)

	for index, item := range videoExt {
		videoExtMap[item] = index
	}

	for index, item := range imageExt {
		imageExtMap[item] = index
	}

	return videoExtMap, imageExtMap
}

func main() {

	videoExtMap, imageExtMap := PreMap()

	root := ""
	desRoot := ""

	var files []string
	var videoFiles []string
	var imageFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			log.Fatal(err)
		}

		if !info.IsDir() {

			extSp := strings.Split(info.Name(), ".")
			extName := extSp[len(extSp)-1]

			if _, ok := videoExtMap[extName]; ok {
				videoFiles = append(videoFiles, path)
			} else if _, ok := imageExtMap[extName]; ok {
				imageFiles = append(imageFiles, path)
			} else {
				files = append(files, path)
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	pathMap := make(map[string][]string)
	pathMap["video"] = videoFiles
	pathMap["image"] = imageFiles
	pathMap["other"] = files

	fmt.Println(pathMap)

	for k, v := range pathMap {
		folderPath := path.Join(desRoot, k)

		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			os.Mkdir(folderPath, os.ModePerm)
		}

		for _, file := range v {
			MoveFile(file, folderPath)
		}

	}
}

// MoveFile 移动文件
func MoveFile(file string, despath string) {
	fileExtSp := strings.Split(file, "/")
	name := fileExtSp[len(fileExtSp)-1]

	filename := FilenameCheck(name, despath)
	err := os.Rename(file, path.Join(despath, filename))

	if err != nil {
		log.Fatal(err)
	}
}

// FilenameCheck 检查并重置路径名称
func FilenameCheck(filename string, desRoot string) string {
	if _, err := os.Stat(path.Join(desRoot, filename)); os.IsNotExist(err) {
		return filename
	}
	newName := uuid.New().String() + filename
	return FilenameCheck(newName, desRoot)
}
