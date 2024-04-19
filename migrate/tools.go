package migrate

import (
	"io"
	"os"
)

// CheckFileExist 判断文件是否存在
func CheckFileExist(filename string) (exists bool) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exists = false
	} else {
		exists = true
	}
	return
}

// WriteFile 写入文件
func WriteFile(filename, content string) (n int, err error) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()

	n, err = f.Write([]byte(content))
	if err == nil && n < len(content) {
		err = io.ErrShortWrite
	}
	return
}
