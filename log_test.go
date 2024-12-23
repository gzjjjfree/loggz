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
	var (
		i = 1
		k = 10
		closeDone chan struct{} = make(chan struct{})
		wg sync.WaitGroup
	)
	//wg.Add(1)
	go func() {
		t := 0
		defer wg.Done()
		for {
			select{
			case <-closeDone:
				fmt.Println("for 循环了 ", t, " 次")
				return				
			case <-time.After(time.Millisecond * 100):
				t++
				//fmt.Println("for 循环了 ", t, " 次")
			}
		}
		
		
	}()
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
	wg.Add(1)
	
	close(closeDone)
	wg.Wait()
	//closeDone <- struct{}{}
	//loggz.Close()
}

func BenchmarkLog(b *testing.B) {
	loggz.Setloglevel(0)
	//defer loggz.Close()
	for i := 0; i < b.N; i++ {
        loggz.WriteTraceLog("模拟写入日志 WriteTraceLog 第 "+strconv.Itoa(i)+" 次")
		loggz.WriteFatalLog(fmt.Sprint("模拟写入日志 WriteFatalLog 第 ", i, " 次"))
		loggz.WriteWarnLog(fmt.Sprint("模拟写入日志 WriteWarnLog 第 ", i, " 次"))
		loggz.WriteDebugLog(fmt.Sprint("模拟写入日志 WriteDebugLog 第 ", i, " 次"))
		loggz.WriteInfoLog(fmt.Sprint("模拟写入日志 WriteInfoLog 第 ", i, " 次"))
    }
	//fmt.Println("看什么时候关闭日志")
	//loggz.Close()
}

