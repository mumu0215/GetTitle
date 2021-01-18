package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"src/common"
	"strings"
	"sync"
)
 //routineCountTotal 线程
var(
	splitTool string           //换行符

	mode=flag.Int("m",0,"mode choice,1:parse from url list,-uF Needed;" +
		"2:parse from port scan file,-pF or -xF Needed")
	routineCountTotal=flag.Int("t",15,"thread")
	myProxy=flag.String("p","","proxy")
	urlFileName=flag.String("uF","","url file name")
	portScanFileName=flag.String("pF","","yujian port scan file")
	nmapXmlFileName=flag.String("xF","","nmap output xmlFileName")
)

func init()  {
	//解析命令行，判断参数合法
	flag.Parse()
	if *routineCountTotal<1{
		fmt.Println("Thread must more than one!")
		os.Exit(0)
	}
	if *mode==0{
		fmt.Println("Mode choice Needed!")
		os.Exit(0)
	}else{
		switch *mode {
		case 1:
			if *urlFileName==""{
				fmt.Println("url FileName input Needed!")
				os.Exit(0)
			}
		case 2:
			if *portScanFileName=="" && *nmapXmlFileName==""{
				fmt.Println("portScan filename or xml filename at least need one!")
				os.Exit(0)
			}
		default:
			fmt.Println("Wrong mode Number! PLZ choose 1 or 2!")
			os.Exit(0)
		}
	}

	//根据系统设置换行符
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
}

func getUrlFileToList(fileName string) []string {
	dataByte,err:=ioutil.ReadFile(fileName)
	if err!=nil{
		fmt.Println("fail to open file "+fileName)
		os.Exit(0)
	}
    data:=strings.TrimSpace(string(dataByte))
    return strings.Split(data,splitTool)
}
func main() {
	//flag.Parse()
	client:=&http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}             //复用client
	if *myProxy!=""{                   //设置代理
		proxy := func(_ *http.Request) (*url.URL, error) {
			return url.Parse(strings.TrimSpace(*myProxy))
		}
		client=&http.Client{Transport:&http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:                  proxy,
		}}
	}
	wg:=&sync.WaitGroup{}
	target:=make(chan string)
	result:=make(chan string)

	urlTitleFile,err:=os.OpenFile("urlTitle.txt",os.O_CREATE|os.O_TRUNC|os.O_RDWR,0666)
	if err!=nil{
		fmt.Println("Fail to open file for result")
		os.Exit(0)
	}
	defer urlTitleFile.Close()
	buf:=bufio.NewWriter(urlTitleFile)
	//接受结果，并处理判断信号
	go func() {
		for rep :=range result{
			if rep==""{            //中断信号
				close(result)
			}else {
				//文件处理传出结果
				tempList:=strings.Split(rep,"\t")
				fmt.Fprintf(buf,"%-60s\t%s\t%-20s\t%s"+splitTool,tempList[0],tempList[1],tempList[2],tempList[3])
				buf.Flush()
			}
		}
	}()

	for i:=0;i<*routineCountTotal;i++{
		wg.Add(1)
		go common.GetOne(wg,client,target,result)
	}
	//mode 1
	//接受url文件
	if *mode==1{
		//分发任务
		for _,baseUrl:=range getUrlFileToList(*urlFileName){
			target <-baseUrl
		}
	}
	target<-""   //工作分发结束
	wg.Wait()
	result<-""   //发出结果中断信号
}