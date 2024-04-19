package excel

import (
	"errors"
	"io"

	"github.com/xuri/excelize/v2"
)

//-----------------------使用示例--------------------
//	newExcel := excel.NewExcelRead() 获取Excel实例
//	err := newExcel.OpenFile("/tmp/test.xlsx") 打开文件
//	dataList, err := newExcel.ReadAllRows(0) 获取所有指定sheet所有的数据(按行)
//	dataList, err := newExcel.ReadRows(0, 1000) 获取指定sheet的数据(迭代器读取)
//---------------------- 使用实例----------------------

// Read Read
type Read struct {
	// File Excel文件句柄
	File *excelize.File
	// Sheets 工作表列表
	Sheets []SheetRead
}

// NewExcelRead NewExcelRead
func NewExcelRead() *Read {
	return &Read{}
}

// OpenFile 打开一个文件
func (r *Read) OpenFile(path string) error {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return err
	}
	sheets := file.GetSheetList()
	if len(sheets) <= 0 {
		return errors.New("文件內容爲空")
	}
	sheetsList := make([]SheetRead, len(sheets))
	for k, v := range sheets {
		sheetsList[k] = SheetRead{
			SheetName: v,
		}
	}
	r.Sheets = sheetsList
	r.File = file
	return nil
}

func (r *Read) OpenReader(reader io.Reader) error {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return err
	}
	sheets := file.GetSheetList()
	if len(sheets) <= 0 {
		return errors.New("文件內容爲空")
	}
	sheetsList := make([]SheetRead, len(sheets))
	for k, v := range sheets {
		sheetsList[k] = SheetRead{
			SheetName: v,
		}
	}
	r.Sheets = sheetsList
	r.File = file
	return nil
}

// ReadAllCols 按列获取指定sheet所有单元格数据
func (r *Read) ReadAllCols(sheetIndex int) ([][]string, error) {
	sheetName, err := r.checkSheet(sheetIndex)
	if err != nil {
		return nil, err
	}
	rows, err := r.File.GetCols(sheetName)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ReadAllRows 按行获取指定sheet所有单元格数据
func (r *Read) ReadAllRows(sheetIndex int) ([][]string, error) {
	sheetName, err := r.checkSheet(sheetIndex)
	if err != nil {
		return nil, err
	}
	rows, err := r.File.GetRows(sheetName)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ReadCols 列迭代器
func (r *Read) ReadCols(sheetIndex, limit int) ([][]string, error) {
	sheetName, err := r.checkSheet(sheetIndex)
	if err != nil {
		return nil, err
	}
	cols := r.Sheets[sheetIndex].Cols
	if cols == nil {
		cols, err = r.File.Cols(sheetName)
		if err != nil {
			return nil, err
		}
		r.Sheets[sheetIndex].Cols = cols
	}

	ret := make([][]string, 0)
	for i := 1; i <= limit; i++ {
		if !cols.Next() {
			return ret, nil
		}
		col, err := cols.Rows()
		if err != nil {
			return nil, err
		}
		ret = append(ret, col)
	}

	return ret, nil
}

// ReadRows 行迭代器
func (r *Read) ReadRows(sheetIndex, limit int) ([][]string, error) {
	sheetName, err := r.checkSheet(sheetIndex)
	if err != nil {
		return nil, err
	}
	rows := r.Sheets[sheetIndex].Rows
	if rows == nil {
		rows, err = r.File.Rows(sheetName)
		if err != nil {
			return nil, err
		}
		r.Sheets[sheetIndex].Rows = rows
	}

	ret := make([][]string, 0)
	for i := 1; i <= limit; i++ {
		if !rows.Next() {
			return ret, nil
		}
		row, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		ret = append(ret, row)
	}

	return ret, nil
}

// checkSheet 检查sheet是否有效
func (r *Read) checkSheet(sheetIndex int) (string, error) {
	if sheetIndex >= len(r.Sheets) {
		return "", errors.New("指定的工作表不存在")
	}
	name := r.File.GetSheetName(sheetIndex)
	if name == "" {
		return name, errors.New("指定的工作表不存在")
	}
	return name, nil
}
