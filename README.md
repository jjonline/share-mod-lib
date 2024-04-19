# go常用包mod库

## 一、项目说明

本仓库收录日常使用中提炼的包，启用go mod，使用单仓库多子包模式。

项目依赖尽可能一直保持为最新，故而本mod库不提供版本功能

> 关于go mod单仓库多子包的拓展资料：https://zhuanlan.zhihu.com/p/134184461

## 二、创建子包

假定要新建一个名为`logger`的子包

step1、项目下新建`logger`目录

step2、在`logger`目录下初始化go mod

````
go mod init github.com/jjonline/share-mod-lib/logger
````
>命令执行完毕，将自动在`logger`目录下生成`go.mod`文件

step3、完善子包代码并提交

step4、打tag，tag名称规则：`子包名/v子包版本号`

> 本例发v0.0.1版本，则tag名称为`logger/v0.0.1`

````
# 直接在当前分支下当前commit下打tag
git tag -a logger/v0.0.1 -m "打这个tag的说明"

# 在指定commit下打tag
git tag -a logger/v0.0.1 0a3a52e -m "打这个tag的说明"

# 将本地tag发布到远程仓库
# 推送全部
git push origin --tags
# 指定tag标签
git push origin logger/v0.0.1

# 删除远程tag
git push origin :refs/tags/[tagName]
## 例如删除远程 logger/v1.0.0 这个tag
git push origin :refs/tags/logger/v1.0.0
````

## 三、注意事项

1、所有子包务必添加`README.md`文件，完善子包的使用说明

2、子包本身不要出现过多的依赖，特别是尽量不要依赖项目内其他子包

## 四、本地开发调试

本地开发过程中需使用某个包时，使用`replace`命令

譬如需使用`image`包做调试，在引入项目`go.mod`文件最后一行添加如下代码

> 注意：请将`__DIR__`替换为你真实的项目路径

````
replace github.com/jjonline/share-mod-lib/image => __DIR__/go-lib-backend/image
````