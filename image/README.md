# Image 图像处理包

## 说明

本包用于处理生成接入基于thumbor服务的图片，不直接走gcs流量

## 使用示例

````
// 实例化Thumb图片裁剪+cdn对象
obj := image.NewThumb("https://image.mytvsupper.com", "passowrd")

// 最小尺寸自适应裁剪，生成原图适应的裁剪（非居中裁剪）
url := obj.FitIn().GenUrl("/public/1.jpg", 800, 600) // 自适应裁剪为可能为尺寸800*500

// url := obj.GenUrl(s3存储路径, 宽, 高)
url := obj.GenUrl("/public/1.jpg", 800, 600) // 裁剪为固定尺寸800*600的图并加入cdn缓存

// 如果不需要裁剪，则全部传0即可
url := obj.GenUrl("/public/1.jpg", 0, 0) // 不裁剪原图加入cdn缓存

// 图片质量，用于减少图片大小（0 to 100）
url := obj.FitIn().Quality(90).GenUrl("/public/1.jpg", 800, 600) // 自适应裁剪为可能为尺寸800*500

// 添加水印
// 第一个参数：水印位置 1、2、3、4、5分别代表左上、右上、右下、左下和居中
// 第一个参数：水印图的数字名称
url := obj.Water(1).GenUrl("/public/1.jpg", 0, 0) // 使用默认水印
url := obj.Water(1, 1).GenUrl("/public/1.jpg", 0, 0) // 使用storage目录 water/1.png作为水印logo

````
