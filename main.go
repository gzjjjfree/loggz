package main

import (
	"fmt"
	"time"

	"github.com/gzjjjfree/loggz/log"
)

func main() {
	fmt.Println("Hello, world!")
	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("模拟写入日志第 %v 次！", i)
		log.WriteTraceLog(msg)
	}
	time.Sleep(time.Second *1)
}
