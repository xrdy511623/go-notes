
---
go语言中的锁详解
---

# 1 锁的使用场景
什么时候需要使用锁？
答案是产生数据竞争(data race)时，在并发读写中为了保证数据正确性，需要使用锁，例如多个协程并发读写同一个
string、map、slice、struct等。
使用锁虽然能保证数据的正确性，但是会引起程序性能的大幅退化，因为大量的锁等待和加解锁的开销会耗费大量时间。
那么，如何检测data race呢？

```shell
go build -race xxx.go
go run -race xxx.go
go test -race xxx.go
```
注意:我们可以在单元测试，压力测试或者日常开发调试时使用上述指令检测程序是否存在data race，但在生产环境
千万不能这样做，因为data race检测会带来10倍以上的性能开销，对线上环境影响太大。

# 2 使用锁的最佳实践
要尽量避免使用锁带来的性能退化，我们有以下几个思路：

## 2.1 缩小临界区
在使用锁时，我们为了避免忘记释放锁，一般会使用defer来释放锁，但这样会导致锁的临界区扩大到函数结束；
但如果我们在执行完需要锁保护的操作后(通常是写操作)及时释放锁，便可缩小锁的临界区，提升程序的性能。
当然，前提是我们能保证程序的正确性，譬如在代码比较长的情况下，直接释放锁的话，如果新增了逻辑分支代码
可能会遗漏解锁，此时使用defer会更可靠。


benchmark测试代码详见performance/narrow-critical-space

```shell
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/narrow-critical-space
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkCountDefer-16          1000000000               0.0000145 ns/op               0 B/op          0 allocs/op
BenchmarkCountNarrow-16         1000000000               0.0000114 ns/op               0 B/op          0 allocs/op
PASS
ok      go-notes/lock/performance/narrow-critical-space 0.445s
```
可以看到，在上面这个案例中，缩小临界区，及时释放锁使得程序的性能提升了约21.37%。

## 2.2 减小锁的粒度
具体来讲，就是使用分段锁，将一把全局大锁替换为多个分段锁，减小锁的粒度，这样便能大幅减少锁竞争，通过
空间换时间的方式提升程序性能。

benchmark测试代码详见performance/segment-lock-replace-global-lock

```shell
# 三种场景，分别使用 全局锁 和 分段锁 测试，共 6 个用例。
# 每次测试读写操作合计 10000 次，例如读多写少场景，读 9000 次，写 1000 次。
# 使用 sync.WaitGroup 阻塞直到读写操作全部运行结束。
# 通过benchmark性能对比测试，可以看到:
# 读写比为 9:1 时，分段锁的性能比全局锁性能提升28.5%；
# 读写比为 1:9 时，分段锁和全局锁性能相当；
# 读写比为 5:5 时，分段锁的性能比全局锁性能提升20.9%；
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/segment-lock-replace-global-lock
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkReadMoreLM-16              1302           4905442 ns/op          645666 B/op      38831 allocs/op
BenchmarkReadMoreSM-16              1718           3515601 ns/op          632683 B/op      38701 allocs/op
BenchmarkWriteMoreLM-16             1732           3512546 ns/op          601406 B/op      30708 allocs/op
BenchmarkWriteMoreSM-16             1759           3447555 ns/op          600679 B/op      30701 allocs/op
BenchmarkEqualLM-16                 1417           4349387 ns/op          620102 B/op      34736 allocs/op
BenchmarkEqualSM-16                 1708           3425395 ns/op          616688 B/op      34701 allocs/op
PASS
ok      go-notes/lock/performance/segment-lock-replace-global-lock      39.655s
```

## 2.3 读写分离
在读多写少的场景，采用读写分离对性能提升最为明显，其核心思路是读写和写写是互斥的，但读读可以并发执行，相比
互斥锁所有操作都互斥，读写锁可以减少锁竞争，提升程序性能。

benchmark测试代码详见performance/rw-lock-replace-mutex

```shell
# 三种场景，分别使用 Lock 和 RWLock 测试，共 6 个用例。
# 每次测试读写操作合计 10000 次，例如读多写少场景，读 9000 次，写 1000 次。
# 使用 sync.WaitGroup 阻塞直到读写操作全部运行结束。
# 通过benchmark性能对比测试，可以看到:
# 读写比为 9:1 时，读写锁的性能约为互斥锁的 6.5 倍
# 读写比为 1:9 时，读写锁性能相当
# 读写比为 5:5 时，读写锁的性能约为互斥锁的 2 倍
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/rw-lock-replace-mutex
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkReadMore-16                 100          53971224 ns/op         1246504 B/op      21219 allocs/op
BenchmarkReadMoreRW-16               655           9061475 ns/op         1133192 B/op      20138 allocs/op
BenchmarkWriteMore-16                100          53521364 ns/op         1154239 B/op      20357 allocs/op
BenchmarkWriteMoreRW-16              122          52892140 ns/op         1166760 B/op      20487 allocs/op
BenchmarkEqual-16                     94          55132050 ns/op         1136588 B/op      20173 allocs/op
BenchmarkEqualRW-16                  201          29987758 ns/op         1218998 B/op      21032 allocs/op
PASS
ok      go-notes/lock/performance/rw-lock-replace-mutex 44.083s
```


## 2.4 使用atomic代替锁实现无锁化
如果只是在并发操作时保护一个变量，使用原子操作比使用互斥锁性能更优。
因为互斥锁的实现是通过操作系统来实现的(系统调用), 而atomic原子操作都是通过硬件实现的，效率比前者要高很多。

benchmark测试代码详见performance/atomic-replace-mutex

```shell
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/atomic-replace-mutex
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkAddNormal-16           1000000000               0.0000018 ns/op               0 B/op          0 allocs/op
BenchmarkAddUseAtomic-16        1000000000               0.0000057 ns/op               0 B/op          0 allocs/op
BenchmarkAddUseMutex-16         1000000000               0.0000173 ns/op               0 B/op          0 allocs/op
PASS
ok      go-notes/lock/performance/atomic-replace-mutex  0.356s

```
可以看到，本案例中，使用atomic代替mutex互斥锁，性能可以提升3倍以上。

如果要保护的变量不是int类型，unsafe.Pointer类型，可以使用atomic.Value, atomic.Value可以承载一个
interface{}

# 3 使用锁的避坑指南

## 3.1 锁是不能拷贝的

```shell
grep -h 'must not be copied' $(go env GOROOT)/src/sync/*.go
// A Cond must not be copied after first use.
// noCopy may be embedded into structs which must not be copied
// The zero Map is empty and ready for use. A Map must not be copied after first use.
// A Mutex must not be copied after first use.
// A Once must not be copied after first use.
// A Pool must not be copied after first use.
// A RWMutex must not be copied after first use.
// A WaitGroup must not be copied after first use.
```
可以看到，标准库sync里的数据结构都是不能拷贝的，如果拷贝锁，就是拷贝了状态，等同于使用了新锁，那就是在并发场景
下使用不同的锁来保护全局变量，其结果是无法保证数据的正确性。

譬如下面这个案例：

```golang
package main

import (
	"fmt"
	"sync"
	"time"
)

var num int

func addWrong(m sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	for i := 0; i < 1000; i++ {
		num++
		time.Sleep(time.Microsecond)
	}
}

func addRight(m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	for i := 0; i < 1000; i++ {
		num++
		time.Sleep(time.Microsecond)
	}
}

func main() {
	var m sync.Mutex
	go addWrong(m)
	go addWrong(m)
	//go addRight(&m)
	//go addRight(&m)
	time.Sleep(time.Second * 2)
	fmt.Println("num = ", num)
}
```

如果拷贝锁，使用go vet检测代码会报拷贝锁的提醒
```shell
go vet        
# go-notes/lock/trap/no-copy-of-mutex
./main.go:11:17: addWrong passes lock by value: sync.Mutex
./main.go:31:14: call of addWrong copies lock value: sync.Mutex
./main.go:32:14: call of addWrong copies lock value: sync.Mutex

```
如果执行代码，会发现执行结果与预期不一致，无法保证数据正确性，每次执行结果可能都不一样。





![copy-mutex.png](images%2Fcopy-mutex.png)





解决的方法很简单，不要拷贝锁，传递锁的引用(指针)就好了。





![right-use-of-mutex.png](images%2Fright-use-of-mutex.png)





## 3.2 标准库sync里的锁是不可重入的，所以不要重复加锁，以免造成死锁。

```golang
package main

import (
	"fmt"
	"sync"
)

func HelloWorld(m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	fmt.Println("Hello")
	m.Lock()
	defer m.Unlock()
	fmt.Println("World")
}

func helloWorld(m *sync.Mutex) {
	m.Lock()
	fmt.Println("Hello")
	m.Unlock()
	m.Lock()
	fmt.Println("World")
	m.Unlock()
}

func main() {
	var m sync.Mutex
	HelloWorld(&m)
	//helloWorld(&m)
}

```

执行代码会出现死锁bug。

```shell
 go run main.go
Hello
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [semacquire]:
sync.runtime_SemacquireMutex(0xc000124008, 0x10, 0x1)
        /usr/local/go/src/runtime/sema.go:71 +0x25
sync.(*Mutex).lockSlow(0xc00012a008)
        /usr/local/go/src/sync/mutex.go:138 +0x165
sync.(*Mutex).Lock(...)
        /usr/local/go/src/sync/mutex.go:81
main.HelloWorld(0xc00012a008)
        /Users/qiujun/go/src/go-notes/lock/trap/no-reentry-mutex/main.go:22 +0xed
main.main()
        /Users/qiujun/go/src/go-notes/lock/trap/no-reentry-mutex/main.go:38 +0x2a
exit status 2

```

这里出现死锁的原因在于标准库sync的互斥锁Mutex(包括读写锁)是不可重入的，重复加锁之前这个锁必须是已经释放
了才可以，本案例中释放锁的操作根据defer语法是后进先出(执行)，所以第二次加锁时，第一次加的锁还未释放，
因为它还在等待第二次的defer操作释放锁，而第二次加锁由于第一次的锁还未释放掉所以无法加锁成功，会一直阻塞，
等待第一次锁的释放，最终导致循环等待，出现死锁的bug。

解决的方案是不使用defer，这样便可顺序加锁和释放锁，但是这个问题的关键在于互斥锁Mutex是不可重入的，所以最好
不要重复加锁。

solve-repeat-mutex.png

## 3.3 atomic.Value误用导致程序崩溃
通常我们会使用atomic.Value来确保更新配置的并发安全，但如果我们配置里使用的是无法保证线程安全的map，那么有可能
出现多个协程并发的去读写配置，出现并发读写map的问题导致程序崩溃。
所以，使用atomic.Value需要注意:
虽然atomic.Value可以实现对任何类型(包括自定义类型)数据的原子操作(读写操作)，但是最好不要使用atomic.Value
存储引用类型的值，这样可能会导致数据不是并发安全的。
因为atomic.Value内部实际上维护的是存储值的指针，而这个指针因为不对外暴露，所以认为是并发安全的。然而如果
尝试用它来存储引用类型，维护的就是这个引用类型的指针，则不能保证实际的数据是并发安全的。
对于一个引用类型，我们实际上只是Store了一个指针，只是对一个指针的原子操作，而这个指针实际指向的地址的值，并不在
atomic.Value的保护下，所以并不是并发安全的。

简言之，atomic.Value只保证存取对象时是并发安全的，并不保证存取的对象本身的操作是并发安全的。所以，要么存放
只读对象，要么对象自身的操作集合必须是并发安全的。

另外:
Store写入的数据不能是空指针nil；
对于同一个atomic.Value不能存入不同类型的值。