package image

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// ++++++++++++++++++++++
// 图片裁剪和cdn服务
// 生成1个url，由url所在服务自动检查签名+从s3取文件&裁剪&发布cdn
// ++++++++++++++++++++++

// Thumb 图片裁剪
type Thumb struct {
	host     string // cdn域
	secret   []byte // 密钥
	commands AppliedCommands
}

// AppliedCommands 图片应用命令
type AppliedCommands struct {
	FitInType string
	Resize    string
	Filters   map[string]string
}

// region 实例化一个01的图片裁剪和cdn对象实例

// NewThumb 新建一个Cdn+Thumb图片服务对象
func NewThumb(host, secret string) *Thumb {
	return &Thumb{
		host:   host,
		secret: []byte(secret),
	}
}

// endregion

// region 必须调用的结尾方法，s3路径文件生成完整cdn的Url

// GenUrl 生成thumb图片裁剪服务的后的url
// f2path 存储路径，不带endpoint，例如：/public/images/test.jpg
// width, height，裁剪后的宽高，传0则不裁剪
func (th *Thumb) GenUrl(f2path string, width, height uint32) string {
	if f2path == "" {
		return ""
	}
	var url string
	url = th.resize(width, height).parseUrl(f2path)
	//if width != 0 || height != 0 { //0x0为原图尺寸,减少url长度
	//	url = th.resize(width, height).parseUrl(f2path)
	//} else {
	//	url = th.parseUrl(f2path)
	//}
	return url
}

// endregion

// region 属性方法，在结尾方法前链式调用

// FitIn 链式操作添加属性---按最小尺寸自适应裁剪
func (th *Thumb) FitIn() *Thumb {
	cth := th.clone()
	cth.commands.FitInType = "fit-in"
	return cth
}

// Water 链式操作添加属性---添加水印
//  基于python-thumbor的图片服务经过改造后实现的水印方法，本质上是一个filter
//  URL中表现为：/sec/filters:water(1)/0x0x/PATH.EXT
//  水印图、水印距离边界的偏移量等参数由python-thumbor内置配置不支持自定义
//  - position 指定水印的位置，1-左上角 2-右上角 3-右下角 4-左下角 5-居中，注意：UI给的参数值是2
//  - logo     可选，自定义水印仅支持png格式，水印图文章存储于thumbor概念中Storage的目录为water/1.png water/2.png，传参1、2即可
func (th *Thumb) Water(position uint8, logo ...uint) *Thumb {
	filter := ""
	if len(logo) > 0 {
		filter = fmt.Sprintf("water(%d%d)", position, logo[0])
	} else {
		filter = fmt.Sprintf("water(%d)", position)
	}
	cth := th.clone()
	if len(cth.commands.Filters) == 0 {
		cth.commands.Filters = make(map[string]string, 0)
	}
	cth.commands.Filters["water"] = filter
	return cth
}

// Quality 链式操作添加属性---图片质量（0 to 100）
func (th *Thumb) Quality(quality uint32) *Thumb {
	filter := fmt.Sprintf("quality(%s)", strconv.Itoa(int(quality)))
	cth := th.clone()
	if len(cth.commands.Filters) == 0 {
		cth.commands.Filters = make(map[string]string, 0)
	}
	cth.commands.Filters["quality"] = filter
	return cth
}

// endregion

// region 内部方法

// clone clone
func (th *Thumb) clone() *Thumb {
	cth := &Thumb{
		host:     th.host,
		secret:   th.secret,
		commands: th.commands,
	}
	return cth
}

// resize 固定尺寸裁剪,0x0为原图
func (th *Thumb) resize(width, height uint32) *Thumb {
	cth := th.clone()
	cth.commands.Resize = fmt.Sprintf("%sx%s", strconv.Itoa(int(width)), strconv.Itoa(int(height)))
	return cth
}

// parseUrl 组装url
func (th *Thumb) parseUrl(f2path string) string {
	commands := th.parseCommand()
	path := strings.Trim(f2path, "/")
	if len(commands) != 0 {
		path = commands + "/" + path
	}
	signature := th.sign(path)
	// {baseDomain}/{signature}/{path}
	return fmt.Sprintf("%s/%s/%s", th.host, signature, path)
}

// parseCommand 组装图片应用命令
func (th *Thumb) parseCommand() string {
	var commands []string
	commands = make([]string, 0)
	if len(th.commands.FitInType) > 0 {
		commands = append(commands, th.commands.FitInType)
	}
	if len(th.commands.Resize) > 0 {
		commands = append(commands, th.commands.Resize)
	}
	if len(th.commands.Filters) > 0 {
		var filters string
		for _, v := range th.commands.Filters {
			filters = filters + v + ":"
		}
		commands = append(commands, fmt.Sprintf("filters:%s", strings.Trim(filters, ":")))
	}
	return strings.Join(commands, "/")
}

// sign 签名生成，path eg:100x100/{f2path}
func (th *Thumb) sign(path string) string {
	signature := th.customSha1(path)
	return signature
}

// customSha1 自定义类型sha1摘要计算
func (th *Thumb) customSha1(data string) string {
	sh := hmac.New(sha1.New, th.secret)
	sh.Write([]byte(data))

	dst := base64.URLEncoding.EncodeToString(sh.Sum(nil))
	return strings.Replace(dst, "+/", "-_", -1)
}

// endregion
