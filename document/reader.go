package document

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"os"
	"strings"
)

const maxBytes = 20 << 20 // 20MB

type DocReader struct{}

func NewReader() *DocReader {
	return &DocReader{}
}

func (s *DocReader) ReadFile(filename string) (rows []string, err error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	return s.Read(content)
}

// Read 读取docx文件内容，返回所有行的内容（纯文本）
func (s *DocReader) Read(fileBytes []byte) (rows []string, err error) {
	zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if err != nil {
		return
	}

	// 定位 document.xml 文件并读取其内容
	index := -1
	for i, file := range zipReader.File {
		if file.Name == "word/document.xml" {
			index = i
		}
	}
	if index < 0 {
		err = errors.New("notfound")
		return
	}

	//读取内容
	documentFile, err := zipReader.File[index].Open()
	if err != nil {
		log.Fatal(err)
	}
	defer documentFile.Close()

	content, err := XMLToText(documentFile, []string{"br", "p", "tab"}, []string{"instrText", "script"}, true)
	if err != nil {
		return
	}
	rows = strings.Split(content, "\n")
	return
}

// XMLToText converts XML to plain text given how to treat elements.
func XMLToText(r io.Reader, breaks []string, skip []string, strict bool) (string, error) {
	var result string

	dec := xml.NewDecoder(io.LimitReader(r, maxBytes))
	dec.Strict = strict
	for {
		t, err := dec.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		switch v := t.(type) {
		case xml.CharData:
			result += string(v)
		case xml.StartElement:
			for _, breakElement := range breaks {
				if v.Name.Local == breakElement {
					result += "\n"
				}
			}
			for _, skipElement := range skip {
				if v.Name.Local == skipElement {
					depth := 1
					for {
						t, err := dec.Token()
						if err != nil {
							// An io.EOF here is actually an error.
							return "", err
						}

						switch t.(type) {
						case xml.StartElement:
							depth++
						case xml.EndElement:
							depth--
						}

						if depth == 0 {
							break
						}
					}
				}
			}
		}
	}
	return result, nil
}
