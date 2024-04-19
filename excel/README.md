# Excel处理包

## 说明

本包用于处理Excel文件的导出和读取,使用的Excel处理库为[excelize](https://github.com/xuri/excelize). 文档[地址](https://xuri.me/excelize/zh-hans/)

## 使用示例

Excel文件导出使用示例
````
// 导出模板
exportInfo := excel.ExportInfo{
    Sheets: []excel.SheetWriter{
        {
            Columns: []excel.Columns{
                {Title: "字段1", Width: 10},
                {Title: "字段2", Width: 20},
                {Title: "字段3", Width: 30},
            },
        },
    },
    FileName: "导出列表",
}
// 导出样式
styleID, _ := e.File.NewStyle(&excelize.Style{
    Font: &excelize.Font{
        Size: 13, // 设置字体大小
    },
    Alignment: &excelize.Alignment{
        Horizontal: "center", // 水平剧中
        Vertical:   "center", // 垂直剧中
    },
})
// 获取Excel实例
newExcel := excel.NewExcel()
// 设置导出信息模板
newExcel.SetExportInfo(exportInfo)
// 设置表头样式
newExcel.SetHeaderStyle(styleID)
// 初始化Excel信息(需要提前新建好Excel表头配置)
err := newExcel.InitExcel()

//---------------------迭代器导出---------------------
// 设置Excel文件表头
err = newExcel.SetSheet(true)
// 填充数据(这个是迭代器，需要循环填充数据)
err = newExcel.FillContent(sheetIndex, dataList, writeIndex)
// 释放迭代器
Excel.Flush()
//---------------------迭代器导出---------------------

//---------------------非迭代器导出---------------------
// 设置Excel文件表头
err = newExcel.SetSheet(false)
// 填充数据(这个是非迭代器，调用一次就可以了)
err = newExcel.FillAllContent(sheetIndex, dataList)
//---------------------非迭代器导出---------------------

// 设置response下载
// 防止乱码
fileName := url.QueryEscape(d.Excel.GetFileName())
ctx.Header("Content-Type", "application/octet-stream")
ctx.Header("Content-Disposition", "attachment; filename="+fileName)
ctx.Header("Content-Transfer-Encoding", "binary")
//回写到web 流媒体 形成下载
newExcel.File.Write(ctx.Writer)
````

Excel文件读取使用示例
````
// 获取Excel实例
newExcel := excel.NewExcelRead()
// 打开文件
err := newExcel.OpenFile("/tmp/test.xlsx")
// 按行获取所有指定sheet所有的数据
dataList, err := newExcel.ReadAllRows(0)
// 按行获取d指定sheet的数据(迭代器读取)
dataList, err := newExcel.ReadRows(0, 1000)
// 按列获取所有指定sheet所有的数据
dataList, err := newExcel.ReadAllCols(0)
// 按列获取指定sheet的数据(迭代器读取)
dataList, err := newExcel.ReadCols(0, 1000)
````

## 简单性能测试结果
测试环境基于普通个人计算机
> OS: Ubuntu 20.04.1 LTS  
> CPU: Intel Core i7  
> RAM: 16 GB DDR4  
> HDD: 256GB SSD    
> Go Version: go version go1.15.3 linux/amd64  
> excelize version：v2.3.1  
> 数据库连接数:10  

单表10w条数据,迭代器带样式导出（执行10次，取平均值）

| 每次读取数量 | 文件生成时间 | 输出至浏览器时间 | 整个请求时间 |
|--------|--------|----------|--------|
| 1000   | 9.16s  | 2.52s    | 11.69s |
| 2000   | 7.89s  | 2.39s    | 10.29s |
| 3000   | 7.46s  | 2.42s    | 9.89s  |
| 5000   | 7.22s  | 2.42s    | 9.64s  |

一次性读取,带样式导出（执行10次，取平均值）

| 导出数量 | 文件生成时间 | 输出至浏览器时间 | 整个请求时间 |
|------|--------|----------|--------|
| 1w   | 680ms  | 712ms    | 1.39s  |
| 5w   | 3.70s  | 4.16s    | 7.87s  |
| 10w  | 6.81s  | 7.95s    | 14.76s |

导出表结构
```
CREATE TABLE `ec_excel_test` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `field1` int unsigned NOT NULL DEFAULT '1' COMMENT '字段1',
  `field2` int unsigned NOT NULL DEFAULT '1' COMMENT '字段2',
  `field3` int unsigned NOT NULL DEFAULT '1' COMMENT '字段3',
  `field4` int unsigned NOT NULL DEFAULT '1' COMMENT '字段4',
  `field5` int unsigned NOT NULL DEFAULT '1' COMMENT '字段5',
  `field6` int unsigned NOT NULL DEFAULT '1' COMMENT '字段6',
  `field7` int unsigned NOT NULL DEFAULT '1' COMMENT '字段7',
  `field8` int unsigned NOT NULL DEFAULT '1' COMMENT '字段8',
  `field9` int unsigned NOT NULL DEFAULT '1' COMMENT '字段9',
  `field10` int unsigned NOT NULL DEFAULT '1' COMMENT '字段10',
  `field11` int unsigned NOT NULL DEFAULT '1' COMMENT '字段11',
  `field12` int unsigned NOT NULL DEFAULT '1' COMMENT '字段12',
  `field13` int unsigned NOT NULL DEFAULT '1' COMMENT '字段13',
  `field14` int unsigned NOT NULL DEFAULT '1' COMMENT '字段14',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=100001 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='测试表';
```

读取测试

读取的excel文件为 16列*10w 行,内容为上面表导出的内容(执行10次，取平均值).

* 单Excel文件10w条数据---按行读取----ReadAllRows读取所有:15.70s
* 单Excel文件10w条数据---按列读取----ReadAllCols读取所有:116.9s
---
* 单Excel文件10w条数据---按行读取----ReadRows迭代器读取所有----每次1000:16.50s
* 单Excel文件10w条数据---按行读取----ReadRows迭代器读取所有----每次2000:16.63s
* 单Excel文件10w条数据---按行读取----ReadRows迭代器读取所有----每次5000:16.71s
---

关于Excel库的性能测试数据，可以参考[文档](https://xuri.me/excelize/zh-hans/performance.html)

## 注意事项

**目前迭代器StreamWriterAPI不是线程安全的，所以无法使用goroutine,暂无此项测试数据(给作者提过[issue][1]，作者回复暂时没有计划支持线程安全的迭代器)**

[1]: https://github.com/xuri/excelize/issues/730