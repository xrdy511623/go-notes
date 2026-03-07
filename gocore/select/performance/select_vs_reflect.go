package performance

import "reflect"

// 静态 select vs reflect.Select 性能对比
//
// 标准 select 在编译期确定 case 数量，由编译器生成高效的 selectgo 调用。
// reflect.Select 在运行时动态处理，有反射开销和额外内存分配。

// StaticSelect2 静态 2-case select
func StaticSelect2(n int) int {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	count := 0
	for range n {
		ch1 <- 1
		select {
		case v := <-ch1:
			count += v
		case v := <-ch2:
			count += v
		}
	}
	return count
}

// ReflectSelect2 使用 reflect.Select 实现相同的 2-case select
func ReflectSelect2(n int) int {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch1)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch2)},
	}
	count := 0
	for range n {
		ch1 <- 1
		_, value, _ := reflect.Select(cases)
		count += int(value.Int())
	}
	return count
}

// StaticSelect4 静态 4-case select
func StaticSelect4(n int) int {
	chs := make([]chan int, 4)
	for i := range chs {
		chs[i] = make(chan int, 1)
	}
	count := 0
	for range n {
		chs[0] <- 1
		select {
		case v := <-chs[0]:
			count += v
		case v := <-chs[1]:
			count += v
		case v := <-chs[2]:
			count += v
		case v := <-chs[3]:
			count += v
		}
	}
	return count
}

// ReflectSelect4 使用 reflect.Select 实现相同的 4-case select
func ReflectSelect4(n int) int {
	chs := make([]chan int, 4)
	cases := make([]reflect.SelectCase, 4)
	for i := range chs {
		chs[i] = make(chan int, 1)
		cases[i] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(chs[i]),
		}
	}
	count := 0
	for range n {
		chs[0] <- 1
		_, value, _ := reflect.Select(cases)
		count += int(value.Int())
	}
	return count
}
