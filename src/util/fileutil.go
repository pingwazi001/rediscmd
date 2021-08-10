package util

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//获取可执行文件路径
func ExecFilePath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		path = strings.Replace(path, "\\", "/", -1)
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "", errors.New(`Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}

//删除文件
func RemoveFile(filePath string) {
	os.Remove(filePath)
}

//读取文件作为字符串返回
func ReadFileAsString(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	contentBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(contentBytes), nil
}
