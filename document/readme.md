# document

## 示例：
```
读取doc文件内容：
fileBytes, err := os.ReadFile(...)
rows, err := document.NewReader().Read(fileBytes)
if err != nil {
    return
}
	
导出doc文件：
writer := document.NewWriter()
doc := writer.NewDocument()
doc.AddParagraph().AddRun().AddText("text").SetFontSize(25)
//返回文件bytes
content, err = writer.Output(doc) 
//保存到文件
err = writer.SaveDocx(doc, filename) 
```