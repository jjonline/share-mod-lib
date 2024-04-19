package document

import (
	"encoding/xml"
	"strconv"
)

// Document 表示一个Word文档
// todo 页头页脚
type Document struct {
	XMLName xml.Name `xml:"w:document"`
	XmlWpc  string   `xml:"xmlns:wpc,attr"`
	XmlMc   string   `xml:"xmlns:mc,attr"`
	XmlO    string   `xml:"xmlns:o,attr"`
	XmlR    string   `xml:"xmlns:r,attr"`
	XmlM    string   `xml:"xmlns:m,attr"`
	XmlV    string   `xml:"xmlns:v,attr"`
	XmlWp14 string   `xml:"xmlns:wp14,attr"`
	XmlWp   string   `xml:"xmlns:wp,attr"`
	XmlW10  string   `xml:"xmlns:w10,attr"`
	XmlW    string   `xml:"xmlns:w,attr"`
	XmlW14  string   `xml:"xmlns:w14,attr"`
	XmlW15  string   `xml:"xmlns:w15,attr"`
	XmlWpg  string   `xml:"xmlns:wpg,attr"`
	XmlWpi  string   `xml:"xmlns:wpi,attr"`
	XmlWne  string   `xml:"xmlns:wne,attr"`
	XmlWps  string   `xml:"xmlns:wps,attr"`
	Mc      string   `xml:"mc:Ignorable,attr"`
	Body    *Body    `xml:"w:body"`
}

// Body 表示Word文档中的主体部分
type Body struct {
	Paragraphs []*Paragraph `xml:"w:p"`
}

// Paragraph 表示Word文档中的一个段落
type Paragraph struct {
	PPr  *PPr   `xml:"w:pPr,omitempty"`
	Runs []*Run `xml:"w:r"`
}

// PPr 段落样式
type PPr struct {
	PBdr *PBdr `xml:"w:pBdr,omitempty"`
}

// PBdr 段落边框属性
type PBdr struct {
	Left   *BorderAttr `xml:"w:left,omitempty"`
	Right  *BorderAttr `xml:"w:right,omitempty"`
	Top    *BorderAttr `xml:"w:top,omitempty"`
	Bottom *BorderAttr `xml:"w:bottom,omitempty"`
}

// BorderAttr 边框属性
type BorderAttr struct {
	Val   string `xml:"w:val,attr"`
	Sz    uint   `xml:"w:sz,attr"`
	Space uint   `xml:"w:space,attr"`
	Color string `xml:"w:color,attr"`
}

// Run 表示Word文档中的一个文本段
type Run struct {
	RPr  *RPr   `xml:"w:rPr,omitempty"`
	Text string `xml:"w:t"`
}

// RPr 文本段属性
type RPr struct {
	RFont *RFont      `xml:"w:rFont,omitempty"` // 字体
	Sz    *ValueField `xml:"w:sz,omitempty"`    // 大小：11
	Color *ValueField `xml:"w:color,omitempty"` // 颜色16进制，不带#前缀：0000FF
}

// RFont 文本段字体
type RFont struct {
	Ascii string `xml:"w:ascii,attr"` // 字体：Times New Roman
	HAnsi string `xml:"w:hAnsi,attr"` // 字体：Times New Roman
}

type ValueField struct {
	Val string `xml:"w:val,attr"`
}

// AddParagraph 添加段落
func (s *Document) AddParagraph() *Paragraph {
	p := &Paragraph{}
	s.Body.Paragraphs = append(s.Body.Paragraphs, p)
	return p
}

// AddRun 添加文本段
func (s *Paragraph) AddRun() *Run {
	r := &Run{}
	s.Runs = append(s.Runs, r)
	return r
}

// SetBorder 设置边框 todo 支持传参自定义边框
func (s *Paragraph) SetBorder() *Paragraph {
	border := &BorderAttr{
		Val:   "single",
		Sz:    12,
		Space: 1,
		Color: "auto",
	}
	s.PPr = &PPr{PBdr: &PBdr{
		Left:   border,
		Right:  border,
		Top:    border,
		Bottom: border,
	}}
	return s
}

// AddText 添加文本
func (s *Run) AddText(text string) *Run {
	s.Text = text
	return s
}

// SetFont 设置字体
func (s *Run) SetFont(ascii, hAnsi string) *Run {
	if s.RPr == nil {
		s.RPr = &RPr{
			RFont: nil,
			Sz:    nil,
			Color: nil,
		}
	}
	s.RPr.RFont = &RFont{
		Ascii: ascii,
		HAnsi: hAnsi,
	}
	return s
}

// SetFontSize 设置字体大小
func (s *Run) SetFontSize(sz uint) *Run {
	if s.RPr == nil {
		s.RPr = &RPr{
			RFont: nil,
			Sz:    nil,
			Color: nil,
		}
	}
	s.RPr.Sz = &ValueField{Val: strconv.Itoa(int(sz))}
	return s
}

// SetColor 设置字体颜色
func (s *Run) SetColor(color string) *Run {
	if s.RPr == nil {
		s.RPr = &RPr{
			RFont: nil,
			Sz:    nil,
			Color: nil,
		}
	}
	s.RPr.Color = &ValueField{Val: color}
	return s
}
