package main

import (
	//"fmt"
	"sync"
	//"time"

	//"time"

	"github.com/gzjjjfree/loggz/loggz"
)

var wg sync.WaitGroup

func main() {
	//fmt.Println("Hello, world!")
	//loggz.Setloglevel(3)
	////time.Sleep(time.Second *1)
	//wg.Add(5)
	//go func() {
	//	defer wg.Done()
	//	for i := 0; i < 10010; i++ {
	//		msg := fmt.Sprintf("模拟写入日志 WriteTraceLog 第 %v 次！", i)
	//		loggz.WriteTraceLog(msg)
	//	}
	//}()
	//go func() {
	//	defer wg.Done()
	//	for i := 0; i < 10030; i++ {
	//		msg := fmt.Sprintf("模拟写入日志 WriteDebugLog 第 %v 次！", i)
	//		loggz.WriteDebugLog(msg)
	//	}
	//}()
	//go func() {
	//	defer wg.Done()
	//	for i := 0; i < 10050; i++ {
	//		msg := fmt.Sprintf("模拟写入日志 WriteInfoLog 第 %v 次！", i)
	//		loggz.WriteInfoLog(msg)
	//	}
	//}()
	//go func() {
	//	defer wg.Done()
	//	for i := 0; i < 10040; i++ {
	//		msg := fmt.Sprintf("模拟写入日志 WriteWarnLog 第 %v 次！", i)
	//		loggz.WriteWarnLog(msg)
	//	}
	//}()
	//go func() {
	//	defer wg.Done()
	//	for i := 0; i < 10050; i++ {
	//		//fmt.Printf("模拟写入日志 WriteErrLog 第 %v 次！\n", i)
	//		msg := fmt.Sprintf("模拟写入日志 WriteErrLog 第 %v 次！", i)
	//		loggz.WriteErrLog(msg)
	//	}
	//}()
	////go func() {
	////for i := 0; i < 100; i++ {
	////	msg := fmt.Sprintf("模拟写入日志 WriteFatalLog 第 %v 次！", i)
	////	log.WriteFatalLog(msg)
	////}
	////}()
	//wg.Wait()
	loggz.Close()
	//time.Sleep(time.Second * 1)
}
