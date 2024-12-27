package loggz_test

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/gzjjjfree/loggz"
)

func TestLog(t *testing.T) {
	loggz.Setloglevel(&loggz.LogConfig{Level: 0})
	var (
		i = 1
		k = 10
		wg sync.WaitGroup
	)
	//wg.Add(1)
	//go func() {
	//	t := 0
	//	defer wg.Done()
	//	for {
	//		select{
	//		case <-closeDone:
	//			fmt.Println("for 循环了 ", t, " 次")
	//			return				
	//		case <-time.After(time.Millisecond * 100):
	//			t++
	//			//fmt.Println("for 循环了 ", t, " 次")
	//		}
	//	}
	//	
	//	
	//}()
	for m:=0;m<10;m++ {
		wg.Add(5)
		go func() {
			defer wg.Done()
			for n:=i;n<k;n++ {
				loggz.WriteTraceLog(fmt.Sprint("模拟写入日志 WriteTraceLog 第 ", n, " 次"))
			}
		}()
		go func() {
			defer wg.Done()
			for n:=i;n<k;n++ {
				loggz.WriteDebugLog(fmt.Sprint("模拟写入日志 WriteDebugLog 第 ", n, " 次"))
			}
		}()
		go func() {
			defer wg.Done()
			for n:=i;n<k;n++ {
				loggz.WriteInfoLog(fmt.Sprint("模拟写入日志 WriteInfoLog 第 ", n, " 次"))
			}
		}()
		go func() {
			defer wg.Done()
			for n:=i;n<k;n++ {
				loggz.WriteWarnLog(fmt.Sprint("模拟写入日志 WriteWarnLog 第 ", n, " 次"))
			}
		}()
		go func() {
			defer wg.Done()
			for n:=i;n<k;n++ {
				loggz.WriteFatalLog(fmt.Sprint("模拟写入日志 WriteFatalLog 第 ", n, " 次"))
			}
		}()
	}
	wg.Wait()	
	loggz.Testwg.Wait()
	time.Sleep(time.Second)
}

func BenchmarkLog(b *testing.B) {
	loggz.Setloglevel(
		&loggz.LogConfig{
			Level: 0,
			//MaxSize: 1,
			//MaxEntries: 1000,
			//MaxAge: 1 * time.Hour,
			//MaxAgeSize: 10,
		},
	)
	for i := 0; i < b.N; i++ {
        loggz.WriteTraceLog("模拟写入日志 WriteTraceLog 第 "+strconv.Itoa(i)+" 次")
		loggz.WriteFatalLog(fmt.Sprint("模拟写入日志 WriteFatalLog 第 ", i, " 次"))
		loggz.WriteWarnLog(fmt.Sprint("模拟写入日志 WriteWarnLog 第 ", i, " 次"))
		loggz.WriteDebugLog(fmt.Sprint("模拟写入日志 WriteDebugLog 第 ", i, " 次"))
		loggz.WriteInfoLog(fmt.Sprint("模拟写入日志 WriteInfoLog 第 ", i, " 次"))		
    }
	loggz.Testwg.Wait()
}

