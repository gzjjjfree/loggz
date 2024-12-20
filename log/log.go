package log

import (
	"fmt"
	"os"

	//"sync"
	"time"
)

//type loglevel int

var (
	trace  = 0
	debug  = 1
	info   = 2
	warn   = 3
	logerr = 4
	fatal  = 5
)

var loglevel = trace

func Setloglevel(level int) {
	loglevel = level
}

var logchan chan *string
var logFile *os.File

//var wg sync.WaitGroup // WaitGroup 用于等待 Goroutine 退出

func init() {
	// 创建日志文件
	var err error
	logFile, err = os.OpenFile("app.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("创建日志文件失败: ", err)
	}
	// 定义一个 10000 个字符串指针的通道，用来接收 log 信息，再用协程写入文件
	logchan = make(chan *string, 10)
	RegisterWriteLog()
}

func WriteTraceLog(msg string) {
	if loglevel <= trace {
		logchan <- &msg
	}	
}

func WriteDebugLog(msg string) {
	if loglevel <= debug {
		logchan <- &msg
	}	
}

func WriteInfoLog(msg string) {
	if loglevel <= info {
		logchan <- &msg
	}	
}
func WriteWarnLog(msg string) {
	if loglevel <= warn {
		logchan <- &msg
	}	
}
func WriteErrLog(msg string) {
	if loglevel <= logerr {
		logchan <- &msg
	}	
}
func WriteFatalLog(msg string) {
	if loglevel <= fatal {
		logchan <- &msg
	}	
}

func RegisterWriteLog() {
	fmt.Println("in log.go func RegissterWriteLog")
	timestamp := "[" + time.Now().Format("2006-01-02 15:04:05") + "]: "
	//wg.Add(1) // 增加 WaitGroup 计数器
	go func() {
		// 这里是匿名函数要执行的代码

		//defer wg.Done()       // Goroutine 退出时减少计数器
		defer logFile.Close() // 确保关闭日志文件
		// 无限循环等待写入
		for {
			fmt.Println("等待日志写入")
			select {
			case logMsg, ok := <-logchan:
				if !ok {
					fmt.Println("通道已关闭，退出日志写入")
					return // 通道关闭时退出循环
				}
				if logMsg != nil {
					fmt.Println("写入日志: ", *logMsg)
					_, err := fmt.Fprintln(logFile, timestamp+*logMsg) // 写入日志信息
					if err != nil {
						fmt.Println("写入日志失败:", err)
					}
				}
				//case <-time.After(time.Second * 5): // 每5秒检查一次，防止select一直阻塞
				//	fmt.Println("日志协程空闲")
			}

		}
	}()
}
