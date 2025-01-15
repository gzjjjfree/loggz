// Package loggz 提供了按等级写入日志，并且可以按文件大小及条目数切分文件的功能
package loggz

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

const (
	trace  = iota // 0
	debug         // 1
	info          // 2
	warn          // 3
	logerr        // 4
	fatal         // 5
)

var (
	subDir                      = "logmsg"
	logFileName                 = "trace"
	scanstr                     = "]: 总共第 "
	logBeforeDone chan struct{} = make(chan struct{})
	logAfterDone  chan struct{} = make(chan struct{})
	Signal                      = struct{}{}
	total         int32         = 0
	logBeforeChan chan string   = make(chan string)
	logAfterChan  chan string   = make(chan string)
	logEnable     bool          = false
	fileReName    bool          = false
	writing       bool          = false
	config                      = defaultLogConfig
	logMutex      sync.Mutex
	saveMutex     sync.Mutex
	timestamp     string
	Testwg        sync.WaitGroup
	once          sync.Once
)

// LogConfig 存储日志配置
type LogConfig struct {
	Level      int           // 日志等级, 最小值 0, 最大值 5, 默认 trace 0 级
	MaxSize    int           // 文件大小, 单位 MB, 最小值为1, 最大值为10, 默认为 5
	MaxEntries int           // 最大条目数, 最小值为 1000, 最大值为 100000, 默认为 10000
	MaxAge     time.Duration // 最长保存时间, 默认 7 天
	MaxAgeSize int           // 保存文件的大小, 默认为 20 MB
}

// 设置默认配置
var defaultLogConfig = &LogConfig{
	Level:      trace,
	MaxSize:    5,
	MaxEntries: 10000,
	MaxAge:     7 * 24 * time.Hour, // 默认 7 天
	MaxAgeSize: 20,
}

// 默认开启 trace 级别的日志
func init() {
	fmt.Println("in writeLog ")
	registerWriteLog()
}

// 建立新的日志的写入
func registerWriteLog() {
	// 每次新建日志都重置总计数
	total = 0
	// 只在这处调用 getTotal 确保 total 的正确
	total = getTotal() + 1
	// 启动一组 goroutine 并跟踪它们的完成状态
	go func() {
		// 标记日志为运行时
		logEnable = true
		filePath, err := getPath(logFileName)
		if err != nil {
			fmt.Println("in writeLog getPath(logFileName) err")
		}
		// defer 函数结束后标记完成
		for {
			select {
			case logMsg := <-logAfterChan:
				if logMsg != "" {
					writeLog(logMsg, filePath)

					if fileReName {
						Testwg.Add(1)
						go checkSaveFile(filePath)
						fileReName = false
					}
				}
			case <-logAfterDone:
				// 接收到信号后结束写入等待
				return
			}
		}
	}()
	go func() {
		msg := ""
		for {
			select {
			case logMsg := <-logBeforeChan:
				if logMsg != "" {
					if msg != "" {
						msg += "\n" + logMsg
					} else {
						msg += logMsg
					}

					if !writing {
						logAfterChan <- msg
						msg = ""
					}
				}
			case <-time.After(time.Second):
				if msg != "" && !writing {
					logAfterChan <- msg
					msg = ""
				}
			case <-logBeforeDone:
				return
			}
		}

	}()
}

// getTotal() 根据日志文件确定原本总计数
func getTotal() int32 {
	logMutex.Lock()
	defer logMutex.Unlock()
	filePath, err := getPath(logFileName + ".log")
	if err != nil {
		return 1
	}
	logFile, err := os.Open(filePath)
	if err != nil {
		return 1
	}
	defer logFile.Close()

	fileInfo, err := logFile.Stat()
	if err != nil {
		return 1
	}
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return 1
	}

	reader := bufio.NewReader(logFile)
	offset := int64(1)
	_, err = logFile.Seek(0, io.SeekEnd) // 从文件末尾向前移动
	if err != nil {
		return 0
	}
	for offset <= fileSize {
		_, err = logFile.Seek(-offset, io.SeekCurrent) // 从文件末尾向前移动
		if err != nil {
			return 0
		}

		b, err := reader.ReadByte()
		if err != nil {
			return 0
		}
		if b == ']' {
			line, err := reader.ReadString('[')
			line = "]" + line
			if err != nil && err.Error() != "EOF" {
				fmt.Println("in getTotal reader.ReadStrin err: ", err)
				return 0
			}
			if len(line) > 0 {
				line = line[:len(line)-1]
				if num, _ := extractNumberString(string(line), scanstr); num != 0 {
					return num
				}				
			}
		}
		offset++
		reader.Discard(reader.Buffered())  // 重置位移
	}
	return 0
	// 逐行读取文件，for 循环读取到最后一行
	//scanner := bufio.NewScanner(logFile)
	//var lastLine string
	//for scanner.Scan() {
	//	lastLine = scanner.Text() // 每次扫描都更新 lastLine
	//}
	//
	//num, _ := extractNumberString(lastLine, scanstr)
	//
	//return num
}

func reverseReadFile(filename string, numLines int) error {
	file, err := os.Open(filename)
	if err != nil {
			return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
			return err
	}
	fileSize := fileInfo.Size()
	if fileSize == 0 {
		return nil
	}

	reader := bufio.NewReader(file)
	lines := make([]string, 0, numLines) // 预分配切片，提高效率
	lineCount := 0
	offset := int64(1)

	for lineCount < numLines && offset <= fileSize {
			_, err = file.Seek(-offset, os.SEEK_END) // 从文件末尾向前移动
			if err != nil {
					return err
			}

			b, err := reader.ReadByte()
			if err != nil {
					return err
			}

			if b == '\n' {
					line, err := reader.ReadString('\n')
					if err != nil {
							return err
					}
					//去除行尾的\n
					line = line[:len(line)-1]
					lines = append(lines, line)
					lineCount++
			}
			offset++
	}

	// 处理最后一行
	if offset > fileSize {
		_, err = file.Seek(0, os.SEEK_SET)
		if err != nil {
			return err
		}
		line,err := reader.ReadString('\n')
		if err != nil {
			return err
		}
					//去除行尾的\n
					line = line[:len(line)-1]
					lines = append(lines, line)
	}

	// 倒序输出
	for i := len(lines) - 1; i >= 0; i-- {
			fmt.Println(lines[i])
	}

	return nil
}

// 根据 beforestr 查找紧跟的数字
func extractNumberString(str string, beforestr string) (int32, error) {
	index := strings.LastIndex(str, beforestr)
	if index == -1 || index+len(beforestr) >= len(str) {
		return 0, fmt.Errorf("字符串格式不正确")
	}
	numStr := ""
	for _, r := range str[index+len(beforestr):] {
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

	return int32(num), nil
}

// 取得文件路径
func getPath(fileName string) (string, error) {
	// 构建子目录的完整路径
	dirPath := filepath.Join(".", subDir) // "." 表示当前目录
	// 创建子目录，如果不存在
	err := os.MkdirAll(dirPath, os.ModePerm) // os.ModePerm 设置所有权限
	if err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}
	filePath := filepath.Join(dirPath, fileName)
	return filePath, nil
}

// 写入日志
func writeLog(msg string, filePath string) {
	// 互斥确保每次只有一个写入操作
	logMutex.Lock()
	setWriting(true)

	defer setWriting(false)
	defer logMutex.Unlock()

	logFile, err := os.OpenFile(filePath+".log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644) // 0644 设置文件权限 os.O_RDWR|os.O_CREATE|os.O_APPEND 文件追加写入
	if err != nil {
		fmt.Println("in writeLog os.OpenFile(filePath err")
	}
	defer logFile.Close()

	fileInfo, err := logFile.Stat()
	if err != nil {
		fmt.Println("获取文件信息出错:", err)
		return
	}
	_, err = fmt.Fprintln(logFile, msg) // 写入日志信息
	if err != nil {
		fmt.Printf("写入日志失败第 %v 次: %v\n", total, err)
		return
	} else {
		// 成功写入日志后，判断日志大小，分离日志限制文件过大
		fileSize := fileInfo.Size()
		if fileSize > int64(config.MaxSize*1024*1024) || total > int32(config.MaxEntries) {
			filePath, err := getPath(logFileName)
			if err != nil {
				fmt.Println("in writeLog getPath(logFileName) err")
			}
			Testwg.Wait()
			partFile, err := os.Create(filePath + ".temp")
			if err != nil {
				fmt.Println("in  writeLog os.Create(filePath + t) err")
				return
			}
			defer partFile.Close()
			logFile.Seek(0, io.SeekStart)
			_, err = io.Copy(partFile, logFile)
			if err != nil && err != io.EOF { // io.EOF 是正常的文件末尾
				fmt.Println("err: ", err)
			}
			err = os.Truncate(filePath+".log", 0) // 将文件截断为 0 字节
			if err != nil {
				fmt.Println("清空文件错误:", err)
				return
			}

			fileReName = true
			total = 0
		}
	}
}
func setWriting(c bool) {
	writing = c
}

// 检查保存文件的日志时间和文件大小
func checkSaveFile(filePath string) {
	//fmt.Println("in checkSaveFile")
	saveMutex.Lock()
	defer saveMutex.Unlock()
	defer Testwg.Done()
	saveFile, err := os.OpenFile(filePath+"save.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("in checkSaveFile os.OpenFile(filePath+save.log err != nil")
		return
	}
	defer saveFile.Close()
	srcFile, err := os.Open(filePath + ".temp")
	if err != nil {
		fmt.Println("in checkSaveFile os.Open(filePath+temp err != nil")
		return
	}
	defer srcFile.Close()
	_, err = io.Copy(saveFile, srcFile)
	if err != nil {
		fmt.Println("in checkSaveFile io.Copy(saveFile, srcFile) err != nil")
		return
	}
	srcFile.Close()
	//return
	err = os.Remove(filePath + ".temp")
	if err != nil {
		fmt.Println("in checkSaveFile os.Remove(filePath+temp err != nil")
		return
	}

	reFile, err := os.OpenFile(filePath+"r", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("in checkSaveFile os.OpenFile(filePath+r err")
	}
	defer reFile.Close()
	saveFile.Seek(0, io.SeekStart)
	scanner := bufio.NewScanner(saveFile)
	checkLine := true // 预设检查日志日期
	currentTime := time.Now().UTC()
	var timeString string

	for scanner.Scan() {
		line := scanner.Text()
		if checkLine {
			str := strings.Split(line, "[")
			if len(str) > 1 {
				str = strings.Split(str[1], "]")
				timeString = str[0]
			}
			layout := "2006-01-02 15:04:05 UTC"
			parsedTime, err := time.Parse(layout, timeString)
			if err != nil { // 日志格式不对时，检查下一条
				continue
			}

			expirationTime := parsedTime.Add(config.MaxAge) // 日志时间加上保存期限
			isExpired := expirationTime.Before(currentTime)
			if !isExpired { // 如果前面的日志不过期, 跳过检查后面的日志
				//fmt.Println("checkLine = false")
				checkLine = false
				fmt.Fprintln(reFile, line)
			}
		} else {
			fmt.Fprintln(reFile, line)
		}
	}
	saveFile.Close()

	fileInfo, _ := reFile.Stat() //os.Stat(filePath+"r")
	fileSize := fileInfo.Size()
	os.Remove(filePath + "save.log")
	if fileSize > int64(config.MaxAgeSize*1024*1024) {
		partFile, err := os.Create(filePath + "d")
		if err != nil {
			fmt.Println("in  checkSaveFile os.Create(filePath + d) err")
			return
		}
		defer partFile.Close()

		_, err = io.CopyN(partFile, reFile, fileSize-int64(config.MaxAgeSize*1024*1024))
		if err != nil && err != io.EOF { // io.EOF 是正常的文件末尾
			fmt.Println("err: ", err)
		}

		partFile.Close()
		err = os.Remove(filePath + "d")
		if err != nil {
			fmt.Println("in checkSaveFile os.Remove(filePath + d) err")
		}
	}
	reFile.Close()
	err = os.Rename(filePath+"r", filePath+"save.log")
	if err != nil {
		fmt.Println("in checkSaveFile os.Rename(filePath+r, filePath+save.log) err")
	}
}

// 设置日志等级, 参数为结构体
//
//	&LogConfig{
//		Level      int           // 日志等级, 最小值 0, 最大值 5, 默认 trace 0 级
//		MaxSize    int           // 文件大小, 单位 MB, 最小值为1, 最大值为10, 默认为 5
//		MaxEntries int           // 最大条目数, 最小值为 1000, 最大值为 100000, 默认为10000
//		MaxAge     time.Duration // 最长保存时间, 默认 7 天 7 * 24 * time.Hour
//		MaxAgeSize int           // 保存文件的大小, 默认为 20 MB
//	}
func Setloglevel(options *LogConfig) {
	logMutex.Lock()
	defer logMutex.Unlock()
	if options != nil { // 检查 options 是否为 nil
		if options.Level != config.Level { // 检查 Level 是否设置
			switch options.Level {
			case 0, 1, 2, 3, 4, 5:
				config.Level = options.Level
			case -1:
				Close()
			}
		}
		if options.MaxSize != config.MaxSize { // 检查 MaxSize 是否设置
			if options.MaxSize >= 1 && options.MaxSize <= 10 {
				config.MaxSize = options.MaxSize
			}
		}
		if options.MaxEntries != 0 { // 检查 MaxEntries 是否设置
			if options.MaxEntries >= 1000 && options.MaxEntries <= 100000 {
				config.MaxEntries = options.MaxEntries
			}
		}
		if options.MaxAge != 0 { // 检查 MaxEntries 是否设置
			if options.MaxAge >= time.Hour && options.MaxAge <= 10*365*24*time.Hour {
				config.MaxAge = options.MaxAge
			}
		}
		if options.MaxAgeSize != 0 { // 检查 MaxAgeSiz 是否设置
			if options.MaxAgeSize >= 10 && options.MaxAgeSize <= 100 {
				config.MaxAgeSize = options.MaxAgeSize
			}
		}
	}
	switch config.Level {
	case 0:
		logFileName = "trace"
	case 1:
		logFileName = "debug"
	case 2:
		logFileName = "info"
	case 3:
		logFileName = "warn"
	case 4:
		logFileName = "error"
	case 5:
		logFileName = "fatal"
	}
}

// 关闭文件及通道
func Close() {
	logEnable = false
	once.Do(func() { logBeforeDone <- Signal })
	once.Do(func() { logAfterDone <- Signal })
	// 等待 registerWriteLog 里的协程结束
	Testwg.Wait()
	once.Do(func() { close(logBeforeChan) })
	once.Do(func() { close(logAfterChan) })
	once.Do(func() { close(logBeforeDone) })
	once.Do(func() { close(logAfterDone) })
}

// Example:
// WriteTraceLog("Hello World")
// 写入 Trace 日志
func WriteTraceLog(msg string) {
	if config.Level <= trace && logEnable {
		timestamp = "[" + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
		atomic.AddInt32(&total, 1)
		msg = timestamp + "[Trace]" + msg
		logBeforeChan <- msg
	}
}

// 写入 Debug 日志
func WriteDebugLog(msg string) {
	if config.Level <= debug && logEnable {
		timestamp = "[" + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
		atomic.AddInt32(&total, 1)
		msg = timestamp + "[Debug]" + msg
		logBeforeChan <- msg
	}
}

// 写入 Info 日志
func WriteInfoLog(msg string) {
	if config.Level <= info && logEnable {
		timestamp = "[" + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
		atomic.AddInt32(&total, 1)
		msg = timestamp + "[Info]" + msg
		logBeforeChan <- msg
	}
}

// 写入 Warn 日志
func WriteWarnLog(msg string) {
	if config.Level <= warn && logEnable {
		timestamp = "[" + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
		atomic.AddInt32(&total, 1)
		msg = timestamp + "[Warn]" + msg
		logBeforeChan <- msg
	}
}

// 写入 Err 日志
func WriteErrLog(msg string) {
	if config.Level <= logerr && logEnable {
		timestamp = "[" + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
		atomic.AddInt32(&total, 1)
		msg = timestamp + "[Err]" + msg
		logBeforeChan <- msg
	}
}

// 写入 Fatal 日志
func WriteFatalLog(msg string) {
	if config.Level <= fatal && logEnable {
		timestamp = "[" + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
		atomic.AddInt32(&total, 1)
		msg = timestamp + "[Fatal]" + msg
		logBeforeChan <- msg
	}
}
