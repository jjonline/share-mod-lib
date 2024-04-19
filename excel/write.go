package excel

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"
)

//-----------------------迭代器使用示例--------------------
//	newExcel := excel.NewExcel() 获取Excel实例
//	newExcel.SetExportInfo(exportInfo) 设置导出信息模板
//	newExcel.SetHeaderStyle(styleID) 设置表头样式
//	err := newExcel.InitExcel() 初始化Excel信息(需要提前新建好Excel表头配置)
//	err = newExcel.SetSheet() 设置Excel文件表头
//	err = newExcel.FillContent(sheetIndex, dataList, writeIndex) 填充数据(这个是迭代器，需要循环填充数据)
//	err = newExcel.Flush() 释放迭代器
//---------------------- 迭代器使用示例----------------------

//---------------------简单性能测试结果------------------------
// 测试环境基于普通个人计算机 (OS: Ubuntu 20.04.1 LTS, CPU: Intel Core i7, RAM: 16 GB DDR4, HDD: 256GB SSD
//  Go Version: go version go1.15.3 linux/amd64, excelize version:v2.3.1 数据库连接数:10
// 单表10w条数据-----使用迭代器查询，每次1000条,带样式导出:13.8s
// 单表10w条数据-----使用迭代器查询，每次2000条,带样式导出:12.3s
// 单表10w条数据-----使用迭代器查询，每次3000条,带样式导出:12.1s
// 单表10w条数据-----使用迭代器查询，每次5000条,带样式导出:11.5s

// 目前迭代器StreamWriterAPI不是线程安全的，所以无法使用goroutine,暂无此项测试数据(给作者提过issue，作者回复暂时没有计划支持线程安全的迭代器)
// https://github.com/xuri/excelize/issues/730
//---------------------简单性能测试结果------------------------

// Excel template
type Excel struct {
	// exportInfo excel模板信息
	exportInfo *ExportInfo
	// File Excel文件句柄
	File *excelize.File
	// headerStyleID 表头默认样式ID
	headerStyleID int
	// contentStyleID Excel内容默认样式ID
	contentStyleID int
}

// NewExcelWrite NewExcelWrite
func NewExcelWrite() *Excel {
	return &Excel{}
}

// getDefaultStyle 新建默认样式
func (e *Excel) getDefaultStyle() int {
	styleID, _ := e.File.NewStyle(e.DefaultStyle())
	return styleID
}

func (e *Excel) DefaultStyle() *excelize.Style {
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

// GetHeaderStyle 获取默认的表头样式
func (e *Excel) GetHeaderStyle() int {
	if e.headerStyleID != 0 {
		return e.headerStyleID
	}
	// 新建样式
	styleID := e.getDefaultStyle()
	e.headerStyleID = styleID
	return styleID
}

// SetHeaderStyle 设置默认的表头样式
func (e *Excel) SetHeaderStyle(styleID int) {
	e.headerStyleID = styleID
}

// getContentStyle 获取默认的内容样式
func (e *Excel) getContentStyle() int {
	if e.contentStyleID != 0 {
		return e.contentStyleID
	}
	// 新建样式
	styleID := e.getDefaultStyle()
	e.contentStyleID = styleID
	return styleID
}

// SetContentStyle 设置默认的表头样式
func (e *Excel) SetContentStyle(styleID int) {
	e.contentStyleID = styleID
}

// GetDefaultFont 获取默认字体(默认为苹方)
func (e *Excel) GetDefaultFont() string {
	fontName := e.exportInfo.FontName
	if fontName == "" {
		fontName = "Heiti SC"
	}
	return fontName
}

// SetDefaultFont 设置默认字体
func (e *Excel) SetDefaultFont(font string) {
	e.exportInfo.FontName = font
}

// SetExportInfo 设置导出模板信息
func (e *Excel) SetExportInfo(info *ExportInfo) {
	e.exportInfo = info
}

// GetExportInfo 获取导出模板信息
func (e *Excel) GetExportInfo() *ExportInfo {
	return e.exportInfo
}

// GetFileName 获取导出文件的完整文件名(带后缀)
func (e *Excel) GetFileName() string {
	return e.exportInfo.FileName
}

// SetFileName 設置导出文件的完整文件名(不需要帶後綴)
func (e *Excel) SetFileName(fileName string) {
	e.exportInfo.FileName = fileName + ".xlsx"
}

// InitExcel 初始化excel部分信息
func (e *Excel) InitExcel() (err error) {
	if e.File != nil {
		return
	}
	if e.exportInfo == nil {
		return errors.New("導出内容为空")
	}
	file := excelize.NewFile()
	// UTCZone8Location 東八區時區location
	var (
		UTCZone8         = "Asia/Hong_Kong"
		UTCZone8Location = time.FixedZone(UTCZone8, 8*3600)
		timeDesc         = time.Now().In(UTCZone8Location).Format("20060102150405")
		fileName         = ""
	)
	// 设置导出文件的文件名
	if e.exportInfo.FileName == "" {
		fileName = "導出文件_" + timeDesc
	} else {
		fileName = e.exportInfo.FileName + "_" + timeDesc
	}
	e.exportInfo.FileName = fileName + ".xlsx"
	// 设置文件字体
	fontName := e.GetDefaultFont()
	if fontName != "" {
		file.SetDefaultFont(fontName)
	}
	// 设置默认的工作表名称
	if len(e.exportInfo.Sheets) <= 0 {
		return errors.New("導出工作表信息未設置")
	}
	newSheetName := e.exportInfo.Sheets[0].SheetName
	if newSheetName != "" {
		// 获取默认的sheet工作表
		sheet := file.GetSheetName(file.GetActiveSheetIndex())
		file.SetSheetName(sheet, newSheetName)
	}
	e.File = file

	return
}

// SetSheet 初始化工作表
func (e *Excel) SetSheet(useStreamWriter bool) error {
	// 为0则使用默认的样式
	headerStyleID := e.GetHeaderStyle()
	// 因为迭代器的API不是很完善，所以需要先设置完样式后再创建迭代器
	for k, v := range e.exportInfo.Sheets {
		if v.StreamWriter != nil {
			continue
		}
		if len(v.Columns) < 1 {
			return errors.New("工作表表頭未設置")
		}
		// 默认的sheet名为"SheetX"
		sheetName := "Sheet" + strconv.Itoa(k+1)
		if v.SheetName != "" {
			sheetName = v.SheetName
		}
		index, _ := e.File.GetSheetIndex(sheetName)
		// 不存在则创建
		if index == -1 {
			index, _ = e.File.NewSheet(sheetName)
		}
		// 更新sheet名称
		e.exportInfo.Sheets[k].SheetName = sheetName
		if !useStreamWriter {
			for columnsKey, columnsValue := range v.Columns {
				column, _ := excelize.ColumnNumberToName(columnsKey + 1)
				if columnsValue.Width != 0 {
					// 设置单元格宽度
					_ = e.File.SetColWidth(sheetName, column, column, float64(columnsValue.Width))
				}
			}
		}
	}
	if !useStreamWriter {
		return nil
	}
	for k, v := range e.exportInfo.Sheets {
		// 设置迭代器
		streamWriter, err := e.File.NewStreamWriter(v.SheetName)
		if err != nil {
			return err
		}
		for columnsKey, columnsValue := range v.Columns {
			// 设置宽度
			_ = streamWriter.SetColWidth(columnsKey+1, columnsKey+1, float64(columnsValue.Width))
		}

		headerValues := make([]interface{}, 0, len(v.Columns))
		for _, cellData := range v.Columns {
			headerValues = append(headerValues, excelize.Cell{StyleID: headerStyleID, Value: cellData.Title})
		}
		_ = streamWriter.SetRow("A1", headerValues)

		// 保存迭代器
		e.exportInfo.Sheets[k].StreamWriter = streamWriter
	}
	return nil
}

// FillContent 填充内容
func (e *Excel) FillContent(sheetIndex int, dataList []RowData, writeIndex int) (err error) {
	if len(dataList) <= 0 {
		return
	}
	// 防止range out of index
	if sheetIndex >= len(e.exportInfo.Sheets) {
		return errors.New("導出工作表信息未設置")
	}

	defaultStyleID := e.getContentStyle()

	// 迭代器按行写入
	for k, v := range dataList {
		rowStyleID := defaultStyleID
		if v.StyleID > 0 {
			rowStyleID = v.StyleID
		}

		rowValues := make([]interface{}, 0, len(v.InterfaceList))
		for _, cellData := range v.InterfaceList {
			//单元格样式
			cellStyleID := rowStyleID
			if cellData.StyleID > 0 {
				cellStyleID = cellData.StyleID
			}

			rowValues = append(rowValues, excelize.Cell{StyleID: cellStyleID, Value: cellData.Value})
		}

		if err = e.exportInfo.Sheets[sheetIndex].StreamWriter.SetRow(fmt.Sprintf("A%d", k+writeIndex), rowValues); err != nil {
			return
		}
	}
	return
}

// Flush 释放迭代器
func (e *Excel) Flush() {
	for k := range e.exportInfo.Sheets {
		if e.exportInfo.Sheets[k].StreamWriter == nil {
			continue
		}
		_ = e.exportInfo.Sheets[k].StreamWriter.Flush()
	}
}

// FillAllContent 获取excel，不使用迭代器
func (e *Excel) FillAllContent(sheetIndex int, dataList []RowData) (err error) {
	if len(dataList) <= 0 {
		return nil
	}

	// 防止range out of index
	if sheetIndex >= len(e.exportInfo.Sheets) {
		return errors.New("導出工作表信息未設置")
	}

	sheetName := e.exportInfo.Sheets[sheetIndex].SheetName
	headerStyleID := e.GetHeaderStyle()
	contentStyleID := e.getContentStyle()
	lastColumn, _ := excelize.ColumnNumberToName(len(e.exportInfo.Sheets[sheetIndex].Columns))

	// 设置表头样式
	_ = e.File.SetCellStyle(sheetName, "A1", lastColumn+"1", headerStyleID)
	// 设置内容样式
	_ = e.File.SetColStyle(sheetName, "A:"+lastColumn, contentStyleID)

	for k, v := range dataList {
		rowStyleID := contentStyleID
		if v.StyleID > 0 {
			rowStyleID = v.StyleID
		}

		rowValues := make([]interface{}, 0, len(v.InterfaceList))
		for _, cellData := range v.InterfaceList {
			//单元格样式
			cellStyleID := rowStyleID
			if cellData.StyleID > 0 {
				cellStyleID = cellData.StyleID
			}

			rowValues = append(rowValues, excelize.Cell{StyleID: cellStyleID, Value: cellData.Value})
		}

		if err = e.exportInfo.Sheets[sheetIndex].StreamWriter.SetRow(fmt.Sprintf("A%d", k+2), rowValues); err != nil {
			return
		}
	}

	return nil
}
