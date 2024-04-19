package excel

import "github.com/xuri/excelize/v2"

// region writer用

// Columns 表头
type Columns struct {
	// Width 列宽度,如果为0则自适应(效果不太好,最好自己指定一下宽度)
	Width uint32
	// Title 表头标题
	Title string
}

// ExportInfo 导出需要的数据
type ExportInfo struct {
	// Sheets 工作表
	Sheets []SheetWriter
	// FileName 文件名称,非必填,为空则使用自动生成的名称
	FileName string
	// FontName 字体名称,非必填,例如:Heiti SC(默认为苹方)
	FontName string
}

// SheetWriter 工作表属性
type SheetWriter struct {
	// Columns 表头配置
	Columns []Columns
	// SheetName 非必填，excel页面左下角展示的名称(默认为sheet1,sheet2...)
	SheetName string
	// StreamWriter sheet迭代器,不用填
	StreamWriter *excelize.StreamWriter
}

// RowData 数据列表
type RowData struct {
	// InterfaceList 行数据
	InterfaceList []CellData
	// StyleID 行样式ID
	StyleID int
}

// CellData 一个
type CellData struct {
	StyleID int         // 行样式ID
	Value   interface{} // 值
}

// endregion

// region read用

// SheetRead 工作表属性
type SheetRead struct {
	// SheetName 工作表名称
	SheetName string
	// Cols 列迭代器
	Cols *excelize.Cols
	// Rows 行迭代器
	Rows *excelize.Rows
}

// endregion
