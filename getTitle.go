package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)
 //routineCountTotal 线程
var userAgent ="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.125 Safari/537.36"
var fileName=flag.String("f","","filename")
var routineCountTotal=flag.Int("t",15,"thread")
var splitTool string

func getOne(group *sync.WaitGroup,client *http.Client,baseUrl chan string,rep chan string) {  //处理每个请求
	//baseUrl:="https://blog.csdn.net/iamlihongwei/article/details/78854899"
	//userAgent :="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.125 Safari/537.36"
	//client:=&http.Client{Transport: &http.Transport{
	//	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	//}}
	for url :=range baseUrl{
		if url==""{
			close(baseUrl)
		}else {
			temp,err:=oneWorker(client,url)
			if err!=nil{
				fmt.Println(url+": "+temp)
				//rep <-err.Error()
			}else {
				rep<-temp
			}
		}
	}
	group.Done()
}
func oneWorker(client *http.Client,baseUrl string) (string,error) {
	req, err := http.NewRequest("GET", baseUrl, nil)
	if err!=nil{
		return  "Fail to create request!",err
	}
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Referer", baseUrl)
	res, err := client.Do(req)
	if err!=nil{
		return "Error occur at Sending Request!",err
	}
	defer res.Body.Close()
	//fmt.Println(res.StatusCode)
	//fmt.Println(res.Header["Server"][0])
	server:=""
	if len(res.Header["Server"])!=0{
		server+=res.Header["Server"][0]
	}else {
		server+="NULL_server!"
	}
	doc,err:=goquery.NewDocumentFromReader(res.Body)
	if err!=nil{
		return "Fail to parse response!",err
	}
	title:=strings.TrimSpace(doc.Find("title").First().Text())
	if title==""{
		title+="NULL_title!"
	}
	report:=baseUrl+"\t"+strconv.Itoa(res.StatusCode)+"\t"+server+"\t"+title
	return report,nil
}
func getTxtToList(fileName string) []string {
	dataByte,err:=ioutil.ReadFile(fileName)
	if err!=nil{
		fmt.Println("fail to open file")
		os.Exit(0)
	}
	switch runtime.GOOS {
	case "windows":
		splitTool="\r\n"
	case "linux":
		splitTool="\n"
	case "darwin":
		splitTool="\r"
	default:
		splitTool="\r\n"
	}
    data:=strings.TrimSpace(string(dataByte))
    return strings.Split(data,splitTool)
}
func main() {
	flag.Parse()
	client:=&http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}             //复用client
	wg:=&sync.WaitGroup{}
	target:=make(chan string)
	result:=make(chan string)
	if *fileName==""{
		fmt.Println("FileName input Needed!")
		os.Exit(0)
	}
	if *routineCountTotal<1{
		fmt.Println("Thread must more than one!")
		os.Exit(0)
	}
	fileResult,err:=os.OpenFile("urlTitle.txt",os.O_CREATE|os.O_TRUNC|os.O_RDWR,0666)
	if err!=nil{
		fmt.Println("Fail to open file for result")
		os.Exit(0)
	}
	defer fileResult.Close()
	buf:=bufio.NewWriter(fileResult)
	//接受结果，并处理判断信号
	go func() {
		for rep :=range result{
			if rep==""{            //中断信号
				close(result)
			}else {
				//文件处理传出结果
				//fileResult.WriteString(rep+"\r\n")
				tempList:=strings.Split(rep,"\t")
				fmt.Fprintf(buf,"%-60s\t%s\t%-20s\t%s"+splitTool,tempList[0],tempList[1],tempList[2],tempList[3])
				buf.Flush()
			}
		}
	}()

	for i:=0;i<*routineCountTotal;i++{
		wg.Add(1)
		go getOne(wg,client,target,result)
	}

	//分发任务
	for _,baseUrl:=range getTxtToList(*fileName){
		target <-baseUrl
	}
	target<-""   //工作结束
	wg.Wait()
	result<-""
}