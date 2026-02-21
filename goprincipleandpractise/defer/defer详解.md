
---
defer详解
---

# 1 使用场景
由于defer语句延迟调用的特性，所以defer语句能非常方便的处理资源释放问题。比如：资源清理、文件关闭、解锁、
关闭连接及记录时间等。

# 2 执行时机
在Go语言的函数中return语句在底层并不是原子操作，它分为给返回值赋值和RET指令两步。而defer语句执行的时机就在返回值
赋值操作后，RET指令执行前。
也就是:
> 返回值=x
> 执行defer语句
> ret指令返回x

# 3 行为规则

## 3.1 参数即时求值
```golang
func a() {
	i := 0
	defer fmt.Println(i)
	i++
}
```

defer语句中的fmt.Println()参数i的值在defer出现时就已经确定了，实际上是复制了一份。之后对变量i的修改不会影响fmt.Printtln()
函数的执行，仍然打印0.
注意: 对于指针类型参数，此规则仍然适用，只不过延迟函数的参数是一个地址值，在这种情况下，defer后面的语句对变量的修改可能会影响
延迟函数。

## 3.2 后进先出(LIFO)
defer语句采取后进先出的设计，类似于栈的方式，函数执行时，每遇到一个defer都会把一个函数压入栈中，函数返回前再将函数从栈中
取出执行，最早被压入栈中的函数最晚被执行。
不仅函数正常返回会执行被defer延迟的函数，函数中任意一个return语句、panic语句均会触发延迟函数。

设计defer的初衷是简化函数返回时资源清理的动作，资源往往有依赖顺序，比如先申请A资源，再根据A资源申请B资源，根据B资源申请
C资源，即申请顺序是A->B->C，释放时往往又要反向进行。这就是把defer设计成后进先出的原因。
每申请到一个用完需要释放的资源时，立即定义一个defer来释放资源是一个很好的习惯。

## 3.3 具名返回值交互
定义defer的函数(下称主函数)可能有返回值，返回值可能有名字(具名返回值)，也可能没有名字(匿名返回值)，延迟函数可能会影响返回值。

a 主函数拥有匿名返回值，返回字面值。
一个主函数拥有一个匿名返回值，返回时使用字面值，比如返回1、2、Hello这样的值，这种情况下defer语句是无法操作返回值的。
譬如:

```golang
func foo() {
	var i int
	defer func() {
		i++
    }()  
	return 1
}
```

b 主函数拥有匿名返回值，返回变量
一个主函数拥有一个匿名返回值，返回本地或全局变量，这种情况下defer语句可以引用返回值，但不会改变返回值。
譬如:

```golang
func foo() {
	var i int
	defer func() {
		i++
    }()  
	return i
}
```
上面的函数返回一个局部变量，同时defer函数也会操作这个局部变量。对于匿名返回值来说，可以假定仍然有一个变量存储返回值，假定返回值变量
为anony，则上面的返回语句可以拆分为以下过程:
```shell
anony = i
i++
return anony
```
由于i是整数，会将值复制给anony，所以在defer语句中修改i值，不会对i的副本，也就是函数返回值anony造成影响。

c 主函数拥有具名返回值 
主函数声明语句中带名字的返回值会被初始化为一个局部变量，函数内可以像使用局部变量一样使用该返回值。如果defer语句操作该返回值，
则可能改变返回值。
譬如:
```golang
func foo() (ret int) {
	defer func() {
		ret++
    }()  
	return 0
```

则上面的返回语句可以拆分为以下过程:
```shell
ret = 0
ret++
return ret
所以最后返回1
```

# 4 defer与panic/recover

defer是recover生效的唯一途径。recover只能在被defer的函数中调用，在其他任何地方调用都会返回nil。

**基本的panic恢复模式**
```golang
func safeOperation() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %v", r)
		}
	}()

	// 可能panic的操作
	riskyWork()
	return nil
}
```

**recover只在当前goroutine的defer中生效**

一个goroutine中的defer无法捕获另一个goroutine中的panic：
```golang
func main() {
	defer func() {
		recover() // 无法捕获子goroutine的panic
	}()

	go func() {
		panic("crash") // 这个panic会导致整个程序崩溃
	}()

	time.Sleep(time.Second)
}
```
因此，每个goroutine如果可能panic，都应该在自己内部使用defer+recover：
```golang
go func() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("goroutine recovered: %v\n%s", r, debug.Stack())
		}
	}()
	riskyWork()
}()
```

**嵌套的recover无效**

recover不能通过嵌套调用生效，必须在defer函数中直接调用：
```golang
defer func() {
	func() {
		recover() // 无效！嵌套调用不生效
	}()
}()

defer func() {
	recover() // 有效
}()
```


# 5 常见陷阱

## 5.1 循环中的defer

defer是在函数退出时执行，而不是在代码块退出时。在循环中使用defer可能导致资源堆积，有资源耗尽的风险。

```golang
// bad: 所有文件句柄直到函数返回才关闭，如果文件数过多，有可能耗尽文件句柄，出现 too many open files错误
func processFiles(filenames []string) error {
	for _, name := range filenames {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close() // 不会在每次循环结束时执行！
		// ...
	}
	return nil
}

// good: 提取到独立函数，每次调用结束即释放
func processFiles(filenames []string) error {
	for _, name := range filenames {
		if err := processFile(name); err != nil {
			return err
		}
	}
	return nil
}

func processFile(name string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	// ...
	return nil
}
```

## 5.2 os.Exit不触发defer

`os.Exit()`会立即终止程序，不执行任何defer语句。这是一个常见的生产环境问题：

```golang
func main() {
	f, _ := os.Create("data.txt")
	defer f.Close()       // 不会执行！
	defer f.Sync()        // 不会执行！

	f.WriteString("important data")
	os.Exit(0) // 直接退出，defer全部跳过
}
```
`log.Fatal`底层调用的就是`os.Exit(1)`，同样不会触发defer：
```golang
func main() {
	db := connectDB()
	defer db.Close() // 不会执行！

	if err := doWork(); err != nil {
		log.Fatal(err) // 内部调用os.Exit(1)
	}
}
```
建议：在main函数中避免使用`log.Fatal`，改用`log.Println` + `os.Exit`前手动清理，或者将业务逻辑放到子函数中正常return错误。

## 5.3 写操作的Close错误

对于只读操作，忽略Close的错误通常没问题。但对于写操作（文件写入、网络发送等），Close可能会flush缓冲区并返回错误，
此时忽略这个错误会导致数据丢失：

```golang
// bad: 写文件时忽略Close错误，可能丢失数据
func writeFile(filename string, data []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close() // 如果Close时flush失败，错误被丢弃了

	_, err = f.Write(data)
	return err
}

// good: 通过具名返回值捕获Close错误
func writeFile(filename string, data []byte) (err error) {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = f.Write(data)
	return err
}
```

## 5.4 参数求值 vs 闭包引用

这是defer中最容易混淆的两种行为，对应main.go中的df6、df7、df8三个例子：

```golang
// 直接传参：参数在defer时求值（输出 1）
var a = 1
defer fmt.Println(a) // a的值在此刻复制
a = 2

// 闭包引用：变量在defer执行时求值（输出 1）
i := 0
defer func() {
	fmt.Println(i) // 引用外部变量i，执行时i已经是1
}()
i++

// 闭包+传参：参数在defer时求值（输出 0）
i := 0
defer func(i int) {
	fmt.Println(i) // 这个i是参数副本，值为0
}(i)
i++
```

核心区别：
- **直接传参**：值在defer语句执行时复制，之后的修改不影响。
- **闭包引用**（不传参）：引用的是外部变量本身，defer执行时才读取最新值。
- **闭包+传参**：即使是闭包，如果通过参数传入，也是在defer时复制值。

## 5.5 指针参数

规则3.1提到"参数在defer时确定"，对于指针类型同样适用——但复制的是地址，因此通过指针修改的内容会被defer看到：

```golang
// 对应main.go中的df9
arr := [3]int{1, 2, 3}
defer func(array *[3]int) {
	for i := range array {
		fmt.Println(array[i]) // 输出 10, 2, 3
	}
}(&arr)
arr[0] = 10 // 通过指针可见
```


# 6 性能演进

Go 1.14之前，defer的实现需要在堆上分配`_defer`结构体并挂到goroutine的defer链表上，每次defer调用约有50ns的额外开销。

Go 1.14引入了**open-coded defer**优化：编译器在编译期将defer调用直接内联到函数返回路径中，避免了运行时的堆分配和链表操作。
优化后defer的开销接近于直接函数调用，几乎可以忽略。

```
Go 1.13:  defer 约 35-50 ns/op
Go 1.14+: defer 约 1-2 ns/op（open-coded场景）
```

注意：open-coded defer有一些限制条件，当defer语句出现在循环中、defer数量过多（>8个）时，编译器会退回到旧的堆分配方式。
这也是建议避免在循环中使用defer的另一个原因——不仅有资源堆积风险，还会失去性能优化。


# 7 总结

| 规则 | 要点 |
|------|------|
| 参数求值 | defer语句出现时立即求值，不是执行时 |
| 执行顺序 | 后进先出(LIFO)，类似栈 |
| 具名返回值 | defer可以修改具名返回值，不能修改匿名返回值 |
| panic/recover | recover只能在defer函数中直接调用，且只在当前goroutine生效 |
| 循环陷阱 | defer在函数退出时执行，不是代码块退出时，循环中应提取子函数 |
| os.Exit | 不触发defer，log.Fatal同理 |
| 写操作Close | 不要忽略写操作的Close错误，用具名返回值捕获 |
| 性能 | Go 1.14+ open-coded defer接近零开销，但循环中的defer会退回堆分配 |
