package gdrive

import (
	"context"
	"errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"io"
)

// 常用的mime类型
const (
	MimeTypeFolder      = "application/vnd.google-apps.folder"                                // 文件夹
	MimeTypeSpreadsheet = "application/vnd.google-apps.spreadsheet"                           // google spreadsheet
	MimeTypeSheet       = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" // office excel
	MimeTypeText        = "text/plain"
	MimeTypeImageJpeg   = "image/jpeg"
	MimeTypeImagePng    = "image/png"
)

type Drive struct {
	ctx     context.Context
	service *drive.Service
}

func New(ctx context.Context, credentialsFile string) (*Drive, error) {
	options := []option.ClientOption{
		option.WithScopes(
			drive.DriveScope,
			drive.DriveAppdataScope,
			drive.DriveFileScope,
			drive.DriveMetadataScope,
			drive.DriveMetadataReadonlyScope,
			drive.DrivePhotosReadonlyScope,
			drive.DriveReadonlyScope,
			drive.DriveScriptsScope,
		),
	}

	if credentialsFile == "" {
		//无证书，则尝试获取默认证书（容器实例授权）
		cred, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			return nil, err
		}
		options = append(options, option.WithCredentials(cred))
	} else {
		options = append(options, option.WithCredentialsFile(credentialsFile))
	}

	service, err := drive.NewService(ctx, options...)
	if err != nil {
		return nil, err
	}

	return &Drive{
		ctx:     ctx,
		service: service,
	}, nil
}

// Create 创建文件/目录
// folderID 文件夹id，必传
// filename 文件名，带扩展名，必传
// mimeType 文件类型，必传，见常量定义
// reader 文件内容
func (d *Drive) Create(folderID, filename, mimeType string, reader io.Reader) (file *drive.File, err error) {
	if folderID == "" || filename == "" || mimeType == "" {
		return nil, errors.New("params error")
	}

	f := &drive.File{
		Name:     filename,
		MimeType: mimeType,
		Parents:  []string{folderID},
	}

	if reader == nil {
		return d.service.Files.Create(f).Do()
	}
	return d.service.Files.Create(f).Media(reader).Do()
}

func (d *Drive) Update(fileID, newName, mimeType string, reader io.Reader) (file *drive.File, err error) {
	f := &drive.File{
		MimeType: mimeType,
	}
	if newName != "" {
		f.Name = newName
	}
	if reader == nil {
		return d.service.Files.Update(fileID, f).Do()
	}
	return d.service.Files.Update(fileID, f).Media(reader).Do()
}

func (d *Drive) Get(fileID string) (file *drive.File, err error) {
	return d.service.Files.Get(fileID).Do()
}

func (d *Drive) Delete(fileID string) (err error) {
	return d.service.Files.Delete(fileID).Do()
}

// List 查询列表
// q 查询条件，语法：name=2022-11-15，注意：name是完整匹配(多个查询条件使用and连接)
// 查询语法参考文档：https://developers.google.com/drive/api/guides/search-files
func (d *Drive) List(q string, limit int64) (files []*drive.File, err error) {
	files = make([]*drive.File, 0)

	if limit == 0 {
		limit = 10
	}

	fileListCall := d.service.Files.List()
	if q != "" {
		fileListCall.Q(q)
	}

	fileList, err := fileListCall.PageSize(limit).Do()
	if err != nil {
		return
	}
	if len(fileList.Files) == 0 {
		return
	}
	files = fileList.Files
	return
}
