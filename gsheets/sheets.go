package gsheets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"net/http"
)

// New 获取Sheets实例
// 注意：需要使用google drive客户端先创建一个空spreadsheet文件，得到文件id（spreadsheetID）后方可调用以下方法。
func New(ctx context.Context, credentialsFile string) (*Sheets, error) {
	srv, err := sheets.NewService(ctx,
		option.WithCredentialsFile(credentialsFile),
		option.WithScopes(drive.DriveScope),
	)
	if err != nil {
		return nil, err
	}
	return &Sheets{ctx: ctx, srv: srv}, nil
}

type Sheets struct {
	ctx context.Context
	srv *sheets.Service
}

type SheetTitleItem struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

// AddSheet 添加工作表
func (s *Sheets) AddSheet(spreadsheetID string, sheetName string) (resp *sheets.BatchUpdateSpreadsheetResponse, err error) {
	req := &sheets.Request{
		AddSheet: &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title: sheetName,
			},
		},
	}
	reqs := &sheets.BatchUpdateSpreadsheetRequest{Requests: []*sheets.Request{req}}
	resp, err = s.srv.Spreadsheets.BatchUpdate(spreadsheetID, reqs).Do()
	if err != nil {
		return
	}
	if resp.HTTPStatusCode != http.StatusOK {
		b, _ := json.Marshal(resp)
		err = errors.New(fmt.Sprintf("http error:%d(%s)", resp.HTTPStatusCode, string(b)))
	}
	return
}

// GetSheetList 获取工作表列表
func (s *Sheets) GetSheetList(spreadsheetID string) (sheetTitleList []SheetTitleItem, err error) {
	resp, err := s.srv.Spreadsheets.Get(spreadsheetID).Fields("sheets(properties(sheetId,title))").Do()
	if err != nil {
		return
	}
	if resp.HTTPStatusCode != http.StatusOK {
		b, _ := json.Marshal(resp)
		err = errors.New(fmt.Sprintf("http error:%d(%s)", resp.HTTPStatusCode, string(b)))
	}

	sheetTitleList = make([]SheetTitleItem, 0, len(resp.Sheets))
	for _, v := range resp.Sheets {
		sheetTitleList = append(sheetTitleList, SheetTitleItem{
			ID:    v.Properties.SheetId,
			Title: v.Properties.Title,
		})
	}
	return
}

// Read 读取工作表数据
func (s *Sheets) Read(spreadsheetID string, sheetName string) (data [][]interface{}, err error) {
	resp, err := s.srv.Spreadsheets.Values.Get(spreadsheetID, sheetName).
		Context(s.ctx).Do()
	if err != nil {
		return
	}
	if resp.HTTPStatusCode != http.StatusOK {
		b, _ := json.Marshal(resp)
		err = errors.New(fmt.Sprintf("http error:%d(%s)", resp.HTTPStatusCode, string(b)))
	}
	return resp.Values, nil
}

// Write 向工作表写入数据（追加模式）
func (s *Sheets) Write(spreadsheetID string, sheetName string, data [][]interface{}) (err error) {
	rows := &sheets.ValueRange{
		Values: data,
	}
	resp, err := s.srv.Spreadsheets.Values.Append(spreadsheetID, sheetName, rows).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(s.ctx).Do()
	if err != nil {
		return
	}
	if resp.HTTPStatusCode != http.StatusOK {
		b, _ := json.Marshal(resp)
		err = errors.New(fmt.Sprintf("http error:%d(%s)", resp.HTTPStatusCode, string(b)))
	}
	return
}

// Clear 清空工作表
func (s *Sheets) Clear(spreadsheetID string, sheetName string) (err error) {
	_, err = s.srv.Spreadsheets.Values.Clear(spreadsheetID, sheetName, &sheets.ClearValuesRequest{}).Do()
	return
}
