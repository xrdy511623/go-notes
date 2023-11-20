package main

import (
	"go-notes/go-principle-and-practise/pprof-practise/animal"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"time"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	log.SetOutput(os.Stdout)
	// 限制CPU使用数
	runtime.GOMAXPROCS(1)
	// 开启锁调用跟踪
	runtime.SetMutexProfileFraction(1)
	// 开启阻塞调用跟踪
	runtime.SetBlockProfileRate(1)

	go func() {
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	for {
		for _, v := range animal.AllAnimals {
			v.Live()
		}
		time.Sleep(time.Second)
	}
}
