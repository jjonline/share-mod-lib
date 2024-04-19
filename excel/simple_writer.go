package excel

import (
	"fmt"
	"github.com/xuri/excelize/v2"
	"io"
)

const (
	DefaultFont  = "Heiti SC" // 默认字体
	MaxRowsLimit = 1000000    // 单个工作表最大行数限制100万条记录
)

// SheetData 工作表数据
type SheetData struct {
	Name   string          // 工作表名称
	Header []Columns       // 表头
	Rows   [][]interface{} // 行数据
}

// SheetDataByGetter 工作表数据
type SheetDataByGetter struct {
	Name       string     // 工作表名称
	Header     []Columns  // 表头
	RowsGetter RowsGetter // 行数据读取器
}

// RowsGetter 行数据读取器
type RowsGetter func() [][]interface{}

// writer template
type writer struct {
	file           *excelize.File // Excel文件句柄
	defaultFont    string         // 默认字体
	defaultStyleID int            // 默认样式
}

func NewSimpleWriter() *writer {
	return &writer{
		file:        excelize.NewFile(),
		defaultFont: DefaultFont,
	}
}

// Export 导出excel，使用StreamWriter模式导出，效率高
// 导出数据示例：
//
//	sheetData := excel.SheetData{
//			Name:       "sheet-001",
//			Header:     headerData,
//			Rows: 		rowsData,
//	}
//
// f, err := os.Create("./export1.xlsx")
// err = excel.NewSimpleWriter().Export(f, sheetData)
func (w *writer) Export(wt io.Writer, sheets ...SheetData) (err error) {
	defer func() {
		if err != nil {
			return
		}
		if err = w.file.Close(); err != nil {
			return
		}
	}()

	//默认样式
	if err = w.setDefaultStyle(); err != nil {
		return
	}

	var (
		sheetName string                 // 工作表名
		sw        *excelize.StreamWriter // StreamWriter
	)

	for i, sheetData := range sheets {
		//设置sheet
		_, sheetName, err = w.newSheet(i, sheetData.Name)
		if err != nil {
			return
		}

		//新建sheet writer
		sw, err = w.file.NewStreamWriter(sheetName)
		if err != nil {
			return
		}

		//写入header
		if err = w.writeHeader(sw, sheetData.Header); err != nil {
			return
		}

		//写入内容
		if err = w.writeContent(sw, len(sheetData.Header) > 0, sheetData.Rows); err != nil {
			return
		}

		//使用StreamWriter必须进行flush
		if err = sw.Flush(); err != nil {
			return
		}
	}

	//将第一个工作表设为默认
	w.file.SetActiveSheet(0)

	return w.file.Write(wt)
}

// ExportByGetter 导出，传入数据获取器
// 分页导出数据示例：
//
//	    page, limit := 1, 1000
//		rowsGetter := func() (data [][]interface{}) {
//			if page > 10 {
//				return
//			}
//			data = make([][]interface{}, 0, limit) //查询数据
//			page++
//			return data
//		}
//
//	sheetData := excel.SheetDataByGetter{
//			Name:       "sheet-001",
//			Header:     headerData,
//			RowsGetter: rowsGetter,
//	}
//
// f, err := os.Create("./export1.xlsx")
// err = excel.NewSimpleWriter().ExportByGetter(f, sheetData)
func (w *writer) ExportByGetter(wt io.Writer, sheets ...SheetDataByGetter) (err error) {
	defer func() {
		if err != nil {
			return
		}
		if err = w.file.Close(); err != nil {
			return
		}
	}()

	//设置默认样式
	if err = w.setDefaultStyle(); err != nil {
		return
	}

	var (
		sheetName string                 // 工作表名
		sw        *excelize.StreamWriter // StreamWriter
	)

	for i, sheetData := range sheets {
		//设置sheet
		_, sheetName, err = w.newSheet(i, sheetData.Name)
		if err != nil {
			return
		}

		//新建sheet writer
		sw, err = w.file.NewStreamWriter(sheetName)
		if err != nil {
			return
		}

		//写入header
		if err = w.writeHeader(sw, sheetData.Header); err != nil {
			return
		}

		//写入内容
		if err = w.writeContentByGetter(sw, len(sheetData.Header) > 0, sheetData.RowsGetter); err != nil {
			return
		}

		//使用StreamWriter必须进行flush
		if err = sw.Flush(); err != nil {
			return
		}
	}

	//将第一个工作表设为默认
	w.file.SetActiveSheet(0)

	return w.file.Write(wt)
}

// sheetName sheet名称处理
func (w *writer) sheetName(i int, name string) string {
	if name == "" {
		name = fmt.Sprintf("Sheet-%d", i)
	}
	return name
}

// defaultStyle 默认样式
func (w *writer) defaultStyle() *excelize.Style {
	return &excelize.Style{
		Font: &excelize.Font{
			Size: 13, // 设置字体大小
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center", // 水平剧中
			Vertical:   "center", // 垂直剧中
		},
	}
}

// setDefaultStyle 设置默认样式
func (w *writer) setDefaultStyle() (err error) {
	//默认字体
	if err = w.file.SetDefaultFont(w.defaultFont); err != nil {
		return
	}

	//默认样式
	styleID, err := w.file.NewStyle(w.defaultStyle())
	if err != nil {
		return
	}
	w.defaultStyleID = styleID
	return
}

// newSheet 新建工作表
func (w *writer) newSheet(seq int, name string) (index int, sheetName string, err error) {
	sheetName = w.sheetName(seq, name)

	//设置sheet
	if seq == 0 {
		index = w.file.GetActiveSheetIndex()
		if err = w.file.SetSheetName(w.file.GetSheetName(index), sheetName); err != nil {
			return
		}
	} else {
		if index, err = w.file.NewSheet(sheetName); err != nil {
			return
		}
	}
	return
}

// writeHeader 写入header
func (w *writer) writeHeader(sw *excelize.StreamWriter, header []Columns) (err error) {
	//设置列宽（根据表头设置）
	for j := range header {
		if header[j].Width > 0 {
			if err = sw.SetColWidth(j+1, j+1, float64(header[j].Width)); err != nil {
				return
			}
		}
	}

	//设置表头
	rowValues := make([]interface{}, 0, len(header))
	for j := range header {
		rowValues = append(rowValues, excelize.Cell{StyleID: w.defaultStyleID, Value: header[j].Title})
	}
	if err = sw.SetRow("A1", rowValues); err != nil {
		return
	}
	return
}

// writeContent 设置内容
func (w *writer) writeContent(sw *excelize.StreamWriter, hasHeader bool, rows [][]interface{}) (err error) {
	currentRow := 1
	if hasHeader {
		currentRow = 2
	}

	total := 0
	for k := range rows {
		if total > MaxRowsLimit {
			break
		}

		//按单元格设置样式（SetRowStyle按行设置不生效）
		rowValues := make([]interface{}, 0, len(rows[k]))
		for l := range rows[k] {
			rowValues = append(rowValues, excelize.Cell{StyleID: w.defaultStyleID, Value: rows[k][l]})
		}
		if err = sw.SetRow(fmt.Sprintf("A%d", currentRow), rowValues); err != nil {
			return
		}

		total++
		currentRow++
	}

	return
}

// writeContentByGetter 设置内容ByGetter
func (w *writer) writeContentByGetter(sw *excelize.StreamWriter, hasHeader bool, rowGetter RowsGetter) (err error) {
	currentRow := 1
	if hasHeader {
		currentRow = 2
	}

	total := 0
	for {
		if total > MaxRowsLimit {
			break
		}

		rows := rowGetter()
		if len(rows) == 0 {
			break
		}

		for k := range rows {
			if total > MaxRowsLimit {
				break
			}

			//按单元格设置样式（SetRowStyle按行设置不生效）
			rowValues := make([]interface{}, 0, len(rows[k]))
			for l := range rows[k] {
				rowValues = append(rowValues, excelize.Cell{StyleID: w.defaultStyleID, Value: rows[k][l]})
			}
			if err = sw.SetRow(fmt.Sprintf("A%d", currentRow), rowValues); err != nil {
				return
			}

			total++
			currentRow++
		}
	}

	return
}
