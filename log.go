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
	subDir                    = "logmsg"
	logFileName               = "trace.log"
	scanstr                   = "]: 总共第 "
	logDone     chan struct{} = make(chan struct{})
	Signal                    = struct{}{}
	total                     = 0
	logchan     chan *string  = make(chan *string, 100)
	logEnable   bool          = false
	config                    = defaultLogConfig
	logFile     *os.File
	logMutex    sync.Mutex
	saveMutex   sync.Mutex
	timestamp   string
	wg          sync.WaitGroup
	Testwg      sync.WaitGroup
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
	reLoadFile(subDir, logFileName)
	registerWriteLog()
}

// 建立新的日志的写入
func registerWriteLog() {
	// 每次新建日志都重置总计数
	total = 0
	// 只在这处调用 getTotal 确保 total 的正确
	total = getTotal() + 1
	wg.Add(1)
	// 启动一组 goroutine 并跟踪它们的完成状态
	go func() {
		// 标记日志为运行时
		logEnable = true
		// 这里是匿名函数要执行的代码
		msg := new(string)
		// defer 函数结束后标记完成
		defer wg.Done()
		for {
			select {
			case logMsg := <-logchan:
				if logMsg != nil {
					// 定义时间格式及计数
					timestamp = "[" + time.Now().UTC().Format("2006-01-02 15:04:05 UTC") + "]: 总共第 " + fmt.Sprint(total) + " 次： "
					*msg = timestamp + *logMsg
					writeLog(*msg)
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

	num, _ := extractNumberString(lastLine, scanstr)

	return num
}

// 根据 beforestr 查找紧跟的数字
func extractNumberString(str string, beforestr string) (int, error) {
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

	return num, nil
}

// 写入日志
func writeLog(msg string) {
	// 互斥确保每次只有一个写入操作
	logMutex.Lock()
	defer logMutex.Unlock()
	total++

	fileInfo, err := logFile.Stat()
	if err != nil {
		fmt.Println("获取文件信息出错:", err)
		reLoadFile(subDir, logFileName)
		return
	}
	_, err = fmt.Fprintln(logFile, msg) // 写入日志信息
	if err != nil {
		fmt.Printf("写入日志失败第 %v 次: %v\n", total, err)
		reLoadFile(subDir, logFileName)
		return
	} else {
		// 成功写入日志后，判断日志大小，分离日志限制文件过大
		filesize := fileInfo.Size()
		if filesize > int64(config.MaxSize*1024*1024) || total > config.MaxEntries {
			logFile.Close()
			logFile = nil
			rename(logFileName)
			reLoadFile(subDir, logFileName)
			total = 0
		}
	}
}

// 重新打开日志文件
func reLoadFile(subDir string, reLoadName string) error {
	if logFile != nil { // 添加空指针检查
		logFile.Close() // 显式关闭文件
		logFile = nil   // 防止重复关闭
	}
	var err error
	// openMode 打开文件的模式
	openMode := os.O_RDWR | os.O_CREATE | os.O_APPEND
	logFile, err = openOrCreateFile(subDir, reLoadName, openMode)
	if err != nil {
		fmt.Println("打开文件失败: ", err)
		return err
	}
	return nil
}

// 设置日志等级, 参数为结构体
//
//		&LogConfig{
//			Level      int           // 日志等级, 最小值 0, 最大值 5, 默认 trace 0 级
//			MaxSize    int           // 文件大小, 单位 MB, 最小值为1, 最大值为10, 默认为 5
//			MaxEntries int           // 最大条目数, 最小值为 1000, 最大值为 100000, 默认为 10000
//			MaxAge     time.Duration // 最长保存时间, 默认 7 天 7 * 24 * time.Hour
//	        MaxAgeSize int           // 保存文件的大小, 默认为 20 MB
//		}
func Setloglevel(options *LogConfig) {
	logMutex.Lock()
	defer logMutex.Unlock()
	if options != nil { // 检查 options 是否为 nil
		if options.Level != config.Level { // 检查 Level 是否设置
			switch options.Level {
			case 0, 1, 2, 3, 4, 5:
				config.Level = options.Level
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
		logFileName = "trace.log"
	case 1:
		logFileName = "debug.log"
	case 2:
		logFileName = "info.log"
	case 3:
		logFileName = "warn.log"
	case 4:
		logFileName = "error.log"
	case 5:
		logFileName = "fatal.log"
	}
	// 更改设置后，重新打开日志文件
	reLoadFile(subDir, logFileName)
}

// 根据目录、文件名、模式打开文件
func openOrCreateFile(subDir string, fileName string, openMode int) (*os.File, error) {
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

// 当满足条件时, 分离文件
func rename(filename string) {
	subPath := filepath.Join(".", subDir)                 // 日志保存目录
	workPath := filepath.Join(subPath, filename)          // 现工作文件
	renewPath := filepath.Join(subPath, "renew"+filename) // 需要把现有内容添加的文件

	openMode := os.O_RDWR | os.O_APPEND
	renewFile, err := os.OpenFile(renewPath, openMode, 0644)
	if err != nil { // 没有添加的目标文件，直接改名
		fmt.Println("in rename os.OpenFile(renewPath, openMode, 0644) is err")
		err1 := os.Rename(workPath, renewPath)
		if err1 != nil {
			// 重命名失败时删除文件，防止日志文件过大
			os.Remove(workPath) // 删除文件
			return
		}
		// 重命名成功的话直接返回
		return
	}
	defer renewFile.Close()
	// 当打开了更新文件时，把工作文件内容添加到更新文件中
	workFile, _ := os.Open(workPath)
	_, err = io.Copy(renewFile, workFile)
	if err != nil {
		return
	}
	workFile.Close()
	os.Remove(workPath)

	fileInfo, err := renewFile.Stat()
	if err != nil {
		fmt.Println("获取文件信息出错:", err)
		defer os.Remove(renewPath)
		return
	}

	fileSize := fileInfo.Size()
	if fileSize > int64(config.MaxSize*1024*1024) {
		// 文件大小超过配置设定, 逐行读取文件, 直至小于设定
		exceedSize := int(fileSize) - config.MaxSize*1024*1024 // 超过的字节数
		renewFile.Seek(0, io.SeekStart)
		scanner := bufio.NewScanner(renewFile)
		var lines []string // 使用切片存储每一行
		for scanner.Scan() && exceedSize > 0 {
			line := scanner.Text()
			lines = append(lines, scanner.Text())
			exceedSize = exceedSize - len(line)
		}
		if config.MaxAge != 0 { // 当有保存时长设定时, 将多出的行添加到保存文件，然后检查它
			savePath := filepath.Join(subPath, "save"+filename) // 根据日期保存的文件，最大不超过 100 MB
			savemsg := strings.Join(lines, "\n")                // 使用换行符连接所有行
			Testwg.Add(1)
			go checkSaveFile(savePath, &savemsg) // 开启协程去检查保存文档
		}
		lines = nil
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		renewFile.Close()
		msg := strings.Join(lines, "\n")
		renewFile, _ = os.OpenFile(renewPath, os.O_WRONLY|os.O_TRUNC, 0644)
		fmt.Fprintln(renewFile, msg)
	}
}

// 检查保存文件的日志时间和文件大小
func checkSaveFile(filePath string, msg *string) {
	saveMutex.Lock()
	defer saveMutex.Unlock()
	saveFile, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	fmt.Fprintln(saveFile, *msg)

	reFile, _ := os.OpenFile(filePath+"r", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
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
				checkLine = false
				fmt.Fprintln(reFile, line)
			}
		} else {
			fmt.Fprintln(reFile, line)
		}
	}
	saveFile.Close()
	reFile.Close()
	time.Sleep(time.Millisecond * 10)
	os.Remove(filePath)
	os.Rename(filePath+"r", filePath)
	fileInfo, _ := os.Stat(filePath)
	fileSize := fileInfo.Size()
	// 当保存文档过大时，分离文档
	if fileSize > int64(config.MaxAgeSize*1024*1024) {
		source, err := os.Open(filePath)
		if err != nil {
			os.Remove(filePath)
		}
		defer source.Close()
		partFile, err := os.Create(filePath + "r")
		if err != nil {
			os.Remove(filePath)
		}
		defer partFile.Close()
		_, err = io.CopyN(partFile, source, fileSize-int64(config.MaxAgeSize*1024*1024))
		if err != nil && err != io.EOF { // io.EOF 是正常的文件末尾
			fmt.Println("err: ", err)
		}
		os.Remove(filePath + "r")
	}
	Testwg.Done()
}

// 关闭文件及通道
func Close() {
	logEnable = false
	close(logchan)
	close(logDone)

	if logFile != nil { // 添加空指针检查
		logFile.Close() // 显式关闭文件
		logFile = nil   // 防止重复关闭
	}
	// 等待 registerWriteLog 里的协程结束
	wg.Wait()
	Testwg.Wait()
}

// Example:
// WriteTraceLog("World")
// 写入 Trace 日志
func WriteTraceLog(msg string) {
	if config.Level <= trace && logEnable {
		logchan <- &msg
	}
}

// 写入 Debug 日志
func WriteDebugLog(msg string) {
	if config.Level <= debug && logEnable {
		logchan <- &msg
	}
}

// 写入 Info 日志
func WriteInfoLog(msg string) {
	if config.Level <= info && logEnable {
		logchan <- &msg
	}
}

// 写入 Warn 日志
func WriteWarnLog(msg string) {
	if config.Level <= warn && logEnable {
		logchan <- &msg
	}
}

// 写入 Err 日志
func WriteErrLog(msg string) {
	if config.Level <= logerr && logEnable {
		logchan <- &msg
	}
}

// 写入 Fatal 日志
func WriteFatalLog(msg string) {
	if config.Level <= fatal && logEnable {
		logchan <- &msg
	}
}
