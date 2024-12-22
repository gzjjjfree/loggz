package loggz

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

var (
	trace                      = 0
	debug                      = 1
	info                       = 2
	warn                       = 3
	logerr                     = 4
	fatal                      = 5
	subDir                     = "logmsg"
	logFileNname               = "trace.log"
	logDone      chan struct{} = make(chan struct{})
	Signal                     = struct{}{}
	total                      = 0
	timestamp    string
	loglevel     = trace
	wg           sync.WaitGroup
	logchan      chan *string
	logFile      *os.File
	logMutex     sync.Mutex
)

// 默认开启 trace 级别的日志
func init() {
	RegisterWriteLog()
}

// 建立新的日志的写入
func RegisterWriteLog() {
	// 每次新建日志都重置总计数
	total = 0
	// 定义一个 100 个字符串指针的通道，用来接收 log 信息，再用协程写入文件
	logchan = make(chan *string, 100)

	var err error
	// openMode 打开文件的模式
	openMode := os.O_RDWR | os.O_CREATE | os.O_APPEND
	logFile, err = openOrCreateFile(subDir, logFileNname, openMode)
	if err != nil {
		fmt.Println("打开文件失败: ", err)
		return
	}

	total = getTotal() + 1

	// 启动一组 goroutine 并跟踪它们的完成状态
	wg.Add(1)
	go func() {
		// 这里是匿名函数要执行的代码
		var msg string
		// defer 函数结束后标记完成
		defer wg.Done()

		for {
			select {
			case logMsg, ok := <-logchan:
				if !ok {
					return // 通道关闭时退出循环
				}
				if logMsg != nil {
					// 定义时间格式及计数
					timestamp = "[" + time.Now().Format("2006-01-02 15:04:05") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
					msg = timestamp + *logMsg
					writeLog(msg)
				}
			case <-logDone:
				// 接收到信号后结束写入等待
				return
			}

		}

	}()
}

// getTotal() 根据日志文件确定原本总计数
func getTotal() int {
	// 逐行读取文件，for 循环读取到最后一行
	scanner := bufio.NewScanner(logFile)
	var lastLine string
	for scanner.Scan() {
		lastLine = scanner.Text() // 每次扫描都更新 lastLine
	}

	num, _ := extractNumberString(lastLine, "]: 总共第 ")

	return num
}

// 根据 beforestr 查找紧跟的数字
func extractNumberString(str string, beforestr string) (int, error) {
	index := strings.LastIndex(str, beforestr)
	if index == -1 || index + len(beforestr) >= len(str) {
		return 0, fmt.Errorf("字符串格式不正确")
	}
	numStr := ""
	for _, r := range str[index + len(beforestr):] {
		if unicode.IsDigit(r) {
			numStr += string(r)
		} else {
			break
		}
	}
	if numStr == "" {
		return 0, fmt.Errorf("没有找到数字")
	}
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("数字转换错误: %w", err)
	}

	return num, nil
}

// 写入日志
func writeLog(msg string) {
	logMutex.Lock()
	total++
	defer logMutex.Unlock()
	_, err := fmt.Fprintln(logFile, msg) // 写入日志信息

	if err != nil {
		fmt.Printf("写入日志失败第 %v 次: %v\n", total, err)
	} else {
		// 成功写入日志后，判断日志大小，分离日志限制文件过大
		fileInfo, err := logFile.Stat()
		if err != nil {
			fmt.Println("获取文件信息出错:", err)
			// 当文件信息出错时，清空文件
			err := logFile.Truncate(0)
			if err != nil {
				logDone <- Signal
				return
			}
			return
		}

		filesize := fileInfo.Size()
		if filesize > 1024*1024*5 || total > 10000 {
			fmt.Println("需要分离日志")
			logFile.Close()
			logFile = nil
			rename(logFileNname)
			var err error
			openMode := os.O_RDWR | os.O_CREATE | os.O_APPEND
			logFile, err = openOrCreateFile(subDir, logFileNname, openMode)
			if err != nil {
				fmt.Println("打开文件失败: ", err)
				logDone <- Signal
				return
			}
			total = 0
		}
	}
}

// 设置日志等级
func Setloglevel(level int) {
	loglevel = level
	switch level {
	case 0:
		logFileNname = "trace.log"
	case 1:
		logFileNname = "debug.log"
	case 2:
		logFileNname = "info.log"
	case 3:
		logFileNname = "warn.log"
	case 4:
		logFileNname = "error.log"
	case 5:
		logFileNname = "fatal.log"
	}
	// 先关闭旧的日志写入，再建立新的
	Close()
	RegisterWriteLog()
}

func openOrCreateFile(subDir, fileName string, openMode int) (*os.File, error) {
	// 构建子目录的完整路径
	dirPath := filepath.Join(".", subDir) // "." 表示当前目录

	// 创建子目录，如果不存在
	err := os.MkdirAll(dirPath, os.ModePerm) // os.ModePerm 设置所有权限
	if err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 构建文件的完整路径
	filePath := filepath.Join(dirPath, fileName)

	// 以读写模式打开文件，如果不存在则创建
	file, err := os.OpenFile(filePath, openMode, 0644) // 0644 设置文件权限 os.O_RDWR|os.O_CREATE|os.O_APPEND 文件追加写入
	if err != nil {
		fmt.Println("打开/创建文件失败:: ", err)
		return nil, err
	}
	
	return file, nil
}

func rename(filename string) {
	oldPath := filepath.Join(".", subDir)
	oldPath = filepath.Join(oldPath, filename)
	
	newPath := filepath.Join(".", subDir)
	newPath = filepath.Join(newPath, "old"+filename)
	
	err := os.Remove(newPath) // 删除文件
	if err != nil {
		fmt.Println("删除文件失败:", err)
	}
	err1 := os.Rename(oldPath, newPath)
	if err1 != nil {
		fmt.Println("重命名文件失败:", err1)
		logDone <- Signal
		return
	}
}

func Close() {
	for {
		if len(logchan) == 0 {
			// 等待通道为了空时，才发送关闭信号
			logDone <- Signal
			if logFile != nil { // 添加空指针检查
				logFile.Close() // 显式关闭文件
				logFile = nil   // 防止重复关闭
			}			
			close(logchan)
			wg.Wait()
			return
		}
	}
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
