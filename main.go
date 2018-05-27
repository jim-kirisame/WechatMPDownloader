package main

import (
	"bufio"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Article
type Article struct {
	Title       string
	Description string
	Author      string
	Content     template.HTML
	Date        time.Time
	TitlePic    string
	OriginalURL string
}

var tplString = `<!DOCTYPE html>
<html>
    <head>
        <meta charset="UTF-8">
        <link href="style.css" rel="stylesheet" type="text/css">
        <title>{{.Title}}</title>
    </head>
    <body>
        <header style="background-image: url({{.TitlePic}})">
            <div class="warpper">
                <div class="title">
                    <h1>{{.Title}}</h1>
                    <h2>{{.Description}}</h2>
                </div>
				<div class="info">
				{{ if .Author}}
					<span class="author">{{.Author}}</span>
				{{ end }}
					<span class="time"><time time="{{.Date}}">{{.Date.Format "2006-01-02"}}</time></span>
				{{ if .OriginalURL}}
                    <span class="origin">
                        <a href="{{.OriginalURL}}">查看原文</a>
					</span>
				{{ end }}
                </div>
            </div>
        </header>
		<div class="container">{{.Content}}</div>
        <footer></footer>
    </body>
</html>`

var usage = `Usage:
WechatMP <URL>          Download an article with the article url.
WechatMP <json-file>    Parse the /mp/profile_ext json file and download all article.
WechatMP <text-file>    Download articles with the multi-line text file.
`

func errHandler(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println(usage)
		return
	}

	tpl, err := template.New("webpage").Parse(tplString)
	errHandler(err)

	checkAndCreateDir("pic")

	if strings.Contains(os.Args[1], "http") {
		url := os.Args[1]
		getHTML(url, tpl)
	} else {
		path := os.Args[1]
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			fmt.Println(path + " is not exist.")
			return
		}
		file, err := os.OpenFile(path, os.O_RDWR, 0755)
		errHandler(err)

		if strings.Contains(path, ".json") {
			urlExp := regexp.MustCompile(`"content_url":"(.*?)"`)
			bytes, err := ioutil.ReadAll(file)
			errHandler(err)

			str := string(bytes)
			str = strings.Replace(str, `\"`, `"`, -1)
			matches := urlExp.FindAllStringSubmatch(str, -1)

			for _, item := range matches {
				url := item[1]
				url = strings.Replace(url, `\`, ``, -1)
				url = html.UnescapeString(url)
				fmt.Println(url)
				getHTML(url, tpl)
			}
		} else {
			reader := bufio.NewReader(file)

			for {
				line, _, err2 := reader.ReadLine()
				if err2 == io.EOF {
					break
				}
				url := string(line)
				fmt.Println(url)
				getHTML(url, tpl)
			}
			err = file.Close()
			errHandler(err)
		}
	}
}

func getHTML(url string, tpl *template.Template) {
	if !strings.Contains(url, "http") {
		return
	}

	resp, err := http.Get(url)
	errHandler(err)

	if resp.StatusCode == 200 {
		robots, err := ioutil.ReadAll(resp.Body)
		errHandler(err)

		err = resp.Body.Close()
		errHandler(err)

		str := string(robots)
		parseMPContent(str, tpl)

	} else {
		fmt.Println(resp.Status)
	}
}

func parseMPContent(content string, tpl *template.Template) {
	imgExp := regexp.MustCompile(`<img[^>]*data-src="(.*?)".*?>`)
	contentExp := regexp.MustCompile(`<div class="rich_media_content " lang=="en" id="js_content">([\S\s]*?)</div>`)
	nicknameExp := regexp.MustCompile(`<span class="rich_media_meta rich_media_meta_text">(.*?)</span>`)
	titleExp := regexp.MustCompile(`var msg_title = "(.*?)";`)
	descriptionExp := regexp.MustCompile(`var msg_desc = "(.*?)";`)
	titlePicExp := regexp.MustCompile(`var msg_cdn_url = "(.*?)";`)
	originalURLExp := regexp.MustCompile(`var msg_source_url = '(.*?)';`)
	timeExp := regexp.MustCompile(`var ct = "([0-9]*)";`)
	backgroundStyleExp := regexp.MustCompile(`url\(&quot;(.*?)&quot;\)`)

	post := Article{}

	contentMatch := contentExp.FindStringSubmatch(content)
	if len(contentMatch) < 2 {
		fmt.Println("Not Original article.")
		return
	}
	innerContent := contentMatch[1]
	post.Description = descriptionExp.FindStringSubmatch(content)[1]
	post.OriginalURL = originalURLExp.FindStringSubmatch(content)[1]
	post.Title = titleExp.FindStringSubmatch(content)[1]
	post.TitlePic = titlePicExp.FindStringSubmatch(content)[1]
	ts := timeExp.FindStringSubmatch(content)[1]
	nickname := nicknameExp.FindStringSubmatch(content)

	if len(nickname) > 0 {
		post.Author = nickname[1]
	}

	timestamp, err := strconv.Atoi(ts)
	errHandler(err)

	post.Date = time.Unix(int64(timestamp), 0)

	imgTags := imgExp.FindAllStringSubmatch(innerContent, -1)
	for _, item := range imgTags {
		newPath := downloadPic(item[1])
		innerContent = strings.Replace(innerContent, item[1], newPath, 1)
	}

	innerContent = strings.Replace(innerContent, "data-src", "src", -1)

	bkgImages := backgroundStyleExp.FindAllStringSubmatch(innerContent, -1)
	for _, item := range bkgImages {
		newPath := downloadPic(item[1])
		innerContent = strings.Replace(innerContent, item[1], newPath, 1)
	}

	post.Content = template.HTML(innerContent)
	post.TitlePic = downloadPic(post.TitlePic)

	file, err := os.Create(post.Date.Format("06-01-02_") + filenameEncode(post.Title) + ".html")
	errHandler(err)

	err = tpl.Execute(file, post)
	errHandler(err)

	err = file.Close()
	errHandler(err)
}

// WxPic
type WxPic struct {
	Type string
	ID   string
	URL  string
}

func downloadPic(url string) string {
	if !strings.Contains(url, "http") {
		return url
	}

	pic := parsePicURL(url)
	var fileName = "pic/" + pic.ID + "." + pic.Type

	resp, err := http.Get(url)
	errHandler(err)

	bytes, err := ioutil.ReadAll(resp.Body)
	errHandler(err)

	err = ioutil.WriteFile(fileName, bytes, 0755)
	errHandler(err)

	return fileName
}

func parsePicURL(url string) WxPic {
	pic := WxPic{URL: url}

	wxfmtExp := regexp.MustCompile(`wx_fmt=[^&]*`)
	match := wxfmtExp.FindStringSubmatch(url)
	if len(match) > 1 {
		pic.Type = match[1]
	} else {
		pic.Type = "jpeg"
	}

	tempString := url[:strings.LastIndex(url, "/")]
	pic.ID = tempString[strings.LastIndex(tempString, "/")+1:]

	return pic
}

func filenameEncode(name string) string {
	name = strings.Replace(name, `\`, `_`, -1)
	name = strings.Replace(name, `/`, `_`, -1)
	name = strings.Replace(name, `*`, `_`, -1)
	name = strings.Replace(name, `:`, `_`, -1)
	name = strings.Replace(name, `?`, `_`, -1)
	name = strings.Replace(name, `|`, `_`, -1)
	name = strings.Replace(name, `"`, `_`, -1)
	name = strings.Replace(name, `<`, `_`, -1)
	name = strings.Replace(name, `>`, `_`, -1)
	return name
}

func checkAndCreateDir(name string) {
	info, err := os.Stat(name)
	if os.IsNotExist(err) || info.IsDir() == false {
		err := os.MkdirAll(name, 0755)
		errHandler(err)
	}
}
