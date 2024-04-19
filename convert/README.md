# convert 

**忽略错误处理**的类型转换工具，一般用于确切的转换场景，请谨慎使用。

* string转数字、数字切片
* 任意类型转string

> 需要捕获转换错误的场景切勿使用该库！

# string转换

## string转数字

````
var str string
var num int

str = "888"
num = convert.String(str).Int()
````

## 任意类型转string

````
var str string
var num int

num = 999
str = convert.IFaceToString(num)
````
