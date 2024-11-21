package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var Client http.Client

// 菜单参数
var Target string
var DorkFile string
var Keyword string
var Token string
var TokenFile string
var NeedWait bool
var NeedWaitSecond int64
var EachWait int64
var OutputFilePath string

// 所有 token 和 dork
var Tokennum = 0
var Tokens []string
var Dorks []string

// 错误次数，超过 100 次就强行结束程序，避免一直运行卡死
var ErrorTimes = 0
var ErrorMaxTimes = 100

// 查询 GitHub API 并处理结果
func query(dork string, token string, outputFile *os.File) {
	// 构造请求
	guri := "https://api.github.com/search/code"
	uri, _ := url.Parse(guri)

	param := url.Values{}
	param.Set("q", dork)
	uri.RawQuery = param.Encode()

	req, _ := http.NewRequest("GET", uri.String(), nil)
	req.Header.Set("accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("User-Agent", "HelloGitHub")

	resp, err := Client.Do(req)

	if err != nil {
		color.Red("error: %v", err)
		return
	}

	defer resp.Body.Close()
	source, _ := ioutil.ReadAll(resp.Body)
	var tmpSource map[string]jsoniter.Any
	_ = jsoniter.Unmarshal(source, &tmpSource)

	// 处理错误或限制
	if tmpSource["documentation_url"] != nil {
		color.Red("error: %s", jsoniter.Get(source, "documentation_url").ToString())
		ErrorTimes++
		if ErrorTimes >= ErrorMaxTimes {
			color.Red("Too many errors, auto stop")
			os.Exit(0)
		}
		if NeedWait {
			color.Blue("We need to wait for %ds", NeedWaitSecond)
			time.Sleep(time.Second * time.Duration(NeedWaitSecond))
			token = getToken()
			query(dork, token, outputFile) // 重试
		}
		return
	}

	// 如果成功获取数据
	if tmpSource["items"] != nil {
		items := tmpSource["items"].GetInterface().([]interface{})
		for _, rawItem := range items {
			// 将 interface{} 转换为 JSON 字节数组
			itemJSON, err := jsoniter.Marshal(rawItem)
			if err != nil {
				color.Red("Error marshaling item: %v", err)
				continue
			}
	
			// 从 JSON 中提取所需字段
			path := jsoniter.Get(itemJSON, "path").ToString()
			htmlURL := jsoniter.Get(itemJSON, "html_url").ToString()
	
			// 实时打印到控制台，包括匹配的关键字
			color.Green("Keyword: %s | Found: %s -> %s", dork, path, htmlURL)
	
			// 实时保存到 CSV 文件
			outputFile.WriteString(fmt.Sprintf("%s,%s\n", path, htmlURL))
		}
	} else {
		color.Blue("No items found for dork: %s", dork)
	}
	
	
}

// 菜单配置
func menu() {
	flag.StringVar(&DorkFile, "gd", "", "github dorks file path")
	flag.StringVar(&Keyword, "gk", "", "github search keyword")
	flag.StringVar(&Token, "token", "", "github personal access token")
	flag.StringVar(&TokenFile, "tf", "", "github personal access token file")
	flag.StringVar(&Target, "target", "", "target which search in github")
	flag.BoolVar(&NeedWait, "nw", true, "if get github api rate limited, need wait ?")
	flag.Int64Var(&NeedWaitSecond, "nws", 20, "how many seconds does it wait each time")
	flag.Int64Var(&EachWait, "ew", 0, "how many seconds does each request should wait ?")
	flag.StringVar(&OutputFilePath, "o", "github_code_results.csv", "output file path for results")

	flag.Usage = func() {
		color.Green(`
	_____ _       _           
	| ____| |_   _(_)___      
	|  _| | \ \ / / / __|
	| |___| |\ V /| \__ \
	|_____|_| \_/ |_|___/

                       v 0.1
`)
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// 检查输入参数
	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(0)
	}
	if Target == "" {
		color.Red("require target")
		os.Exit(0)
	}
	if DorkFile == "" && Keyword == "" {
		color.Red("require keyword or dorkfile")
		os.Exit(0)
	}
	if Token == "" && TokenFile == "" {
		color.Red("require token or tokenfile")
		os.Exit(0)
	}
}

// 解析 token 和 dork 参数
func parseparam() {
	// 解析 token
	if Token != "" {
		Tokens = []string{Token}
	} else if TokenFile != "" {
		tfres, err := ioutil.ReadFile(TokenFile)
		if err != nil {
			color.Red("file error: %v", err)
			os.Exit(0)
		} else {
			tfresLine := strings.Split(string(tfres), "\n")
			for {
				if tfresLine[len(tfresLine)-1] == "" {
					tfresLine = tfresLine[:len(tfresLine)-1]
				} else {
					break
				}
			}
			Tokens = tfresLine
		}
	}
	// 解析 dork
	if Keyword != "" {
		Dorks = []string{Keyword}
	} else if DorkFile != "" {
		dkres, err := ioutil.ReadFile(DorkFile)
		if err != nil {
			color.Red("file error: %v", err)
			os.Exit(0)
		} else {
			dkresLine := strings.Split(string(dkres), "\n")
			for {
				if dkresLine[len(dkresLine)-1] == "" {
					dkresLine = dkresLine[:len(dkresLine)-1]
				} else {
					break
				}
			}
			Dorks = dkresLine
		}
	}
	color.Blue("[+] got %d tokens and %d dorks\n\n", len(Tokens), len(Dorks))
}

// 获取当前 token
func getToken() string {
	token := Tokens[Tokennum]
	Tokennum++
	if len(Tokens) == Tokennum {
		Tokennum = 0
	}
	return token
}

// 主函数
func main() {
	menu()
	parseparam()

	// 打开 CSV 文件
	outputFile, err := os.Create(OutputFilePath)
	if err != nil {
		color.Red("Cannot create file: %v", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	// 写入 CSV 表头
	outputFile.WriteString("File Path,HTML URL\n")

	Client = http.Client{}

	for _, dork := range Dorks {
		token := getToken()
		query(fmt.Sprintf("%s %s", Target, dork), token, outputFile)
		time.Sleep(time.Second * time.Duration(EachWait))
	}
	color.Green("任务完成！")
}
