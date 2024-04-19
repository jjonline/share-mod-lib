package document

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
)

type DocWriter struct{}

func NewWriter() *DocWriter {
	return &DocWriter{}
}

func (s *DocWriter) NewDocument() *Document {
	return &Document{
		XMLName: xml.Name{},
		XmlWpc:  "http://schemas.microsoft.com/office/word/2010/wordprocessingCanvas",
		XmlMc:   "http://schemas.openxmlformats.org/markup-compatibility/2006",
		XmlO:    "urn:schemas-microsoft-com:office:office",
		XmlR:    "http://schemas.openxmlformats.org/officeDocument/2006/relationships",
		XmlM:    "http://schemas.openxmlformats.org/officeDocument/2006/math",
		XmlV:    "urn:schemas-microsoft-com:vml",
		XmlWp14: "http://schemas.microsoft.com/office/word/2010/wordprocessingDrawing",
		XmlWp:   "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing",
		XmlW10:  "urn:schemas-microsoft-com:office:word",
		XmlW:    "http://schemas.openxmlformats.org/wordprocessingml/2006/main",
		XmlW14:  "http://schemas.microsoft.com/office/word/2010/wordml",
		XmlW15:  "http://schemas.microsoft.com/office/word/2012/wordml",
		XmlWpg:  "http://schemas.microsoft.com/office/word/2010/wordprocessingGroup",
		XmlWpi:  "http://schemas.microsoft.com/office/word/2010/wordprocessingInk",
		XmlWne:  "http://schemas.microsoft.com/office/word/2006/wordml",
		XmlWps:  "http://schemas.microsoft.com/office/word/2010/wordprocessingShape",
		Mc:      "w14 w15 wp14",
		Body:    &Body{},
	}
}

// ToXml 将文档序列化为XML格式
func (s *DocWriter) ToXml(doc *Document) ([]byte, error) {
	return xml.Marshal(doc)
}

// SaveDocx 保存docx文件
func (s *DocWriter) SaveDocx(doc *Document, filename string) (err error) {
	content, err := s.Output(doc)
	//content, err := s.ToXml(doc)
	if err != nil {
		return
	}
	return os.WriteFile(filename, content, os.ModePerm)
}

// Output 输出docx文件流
func (s *DocWriter) Output(doc *Document) (fileBytes []byte, err error) {
	xmlBytes, err := s.ToXml(doc)
	if err != nil {
		return
	}

	xmlFile := make(map[string][]byte)
	xmlFile["[Content_Types].xml"] = []byte(XMLHeader + tplContentTypeXML)
	xmlFile["_rels/.rels"] = []byte(XMLHeader + tplRelsXML)
	xmlFile["docProps/app.xml"] = []byte(XMLHeader + tplAppXML)
	xmlFile["docProps/core.xml"] = []byte(XMLHeader + tplCoreXML)
	xmlFile["docProps/custom.xml"] = []byte(XMLHeader + tplCustomXML)
	xmlFile["word/theme/theme1.xml"] = []byte(XMLHeader + tplThemeXML)
	xmlFile["word/endnotes.xml"] = []byte(XMLHeader + tplEndnotesXML)
	xmlFile["word/fontTable.xml"] = []byte(XMLHeader + tplFontTableXML)
	xmlFile["word/footnotes.xml"] = []byte(XMLHeader + tplFootNotesXML)
	xmlFile["word/settings.xml"] = []byte(XMLHeader + tplSettingsXML)
	xmlFile["word/styles.xml"] = []byte(XMLHeader + tplStylesXML)
	xmlFile["word/webSettings.xml"] = []byte(XMLHeader + tplWebSettingsXML)

	buf := bytes.NewBuffer(nil)
	d := zip.NewWriter(buf)
	var writer io.Writer

	for name, content := range xmlFile {
		writer, err = d.Create(name)
		if err != nil {
			return
		}
		if _, err = writer.Write(content); err != nil {
			return
		}
	}

	if writer, err = d.Create(DocumentFileKey); err != nil {
		return
	}
	if _, err = writer.Write([]byte(XMLHeader)); err != nil {
		return
	}
	if _, err = writer.Write(xmlBytes); err != nil {
		return
	}
	if err = d.Close(); err != nil {
		return
	}
	return buf.Bytes(), nil
}
