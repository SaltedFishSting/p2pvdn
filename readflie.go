package main

//import (
//	"bufio"

//	"io/ioutil"
//	"os"
//	"path"
//	"path/filepath"
//	"strconv"
//	"strings"
//)

//func check(e error) error {
//	if e != nil {
//		return e
//	}
//	return nil
//}

//// param file：flag文件的路径 extractor：flag文件对应的promtheus函数
//func fileToPrometheus(file string, extractor VMDExtractor) error {
//	f, err := os.Open(file)
//	check(err)
//	buf := bufio.NewReader(f)
//	line, err := buf.ReadString('\n') //读取flag文件 找到当前写到的txt文件

//	flag := strings.Split(string(line), "|")
//	pathDir := filepath.Dir(file)
//	flietxt := strings.TrimSpace(flag[0])
//	txtf, err := os.Open(path.Join(pathDir, flietxt)) //读取txt文件
//	check(err)
//	txtbuf := bufio.NewReader(txtf)
//	txtstring, err := ioutil.ReadAll(txtbuf)
//	check(err)
//	lines := strings.Split(string(txtstring), "\n")
//	i, err := strconv.Atoi(flag[1])
//	check(err)
//	j, err := strconv.Atoi(flag[2])
//	check(err)
//	//信息传给prometheus
//	for i <= j {
//		infos := strings.Split(string(lines[i-1]), "|")
//		if err := extractor(infos); err != nil {
//			return err
//		}
//		i++
//	}
//	return nil
//}
