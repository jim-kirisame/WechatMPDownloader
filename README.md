# 微信公众号文章下载工具

## 使用方法

### 下载单篇文章

`WechatMPDownloader.exe <URL> ` 

使用例：

`WechatMPDownloader.exe https://mp.weixin.qq.com/s/_Kutnupvx2xEHCX2C6IG7A`

### 下载多个文章

假设你现在目录里面有个叫做links.txt的文件，里面的内容是：

```
https://mp.weixin.qq.com/s/_Kutnupvx2xEHCX2C6IG7A
https://mp.weixin.qq.com/s/Rqcc2IXBsp3k5CDd0P113w
https://mp.weixin.qq.com/s/T23E-jOVyUJsZ3lprUHLCA
```

那么，使用`WechatMPDownloader.exe links.txt`就可以下载上面的三篇文章了。

### 解析并下载多个文章

使用Fiddler抓取公众号的历史消息页面的请求，可以找到几个`GET /mp/profile_ext?action=getmsg`包。将这些包使用Save-Response-Response Body功能保存到本地，可以得到几个json的包。假设你现在已经有了一个包，它的名字是links.json，那么你可以使用这条命令来下载里面的所有文章：

`WechatMPDownloader.exe links.json`

需要注意的是，如果你想下载公众号的所有历史消息，那么你可能需要使用多个包来下载。