package performance

// select case 数量与 default 对性能的影响
//
// select 的 case 数量越多，selectgo 需要遍历的 scase 越多，
// 随机排列和加锁排序的开销也更大。
// 带 default 的 select 会被编译器优化为非阻塞调用，绕过 selectgo。

var selectSink int

// ---------- 带 default 的 select（编译器优化为非阻塞调用） ----------

// SelectDefault1Case 1个 case + default，编译为 selectnbrecv
func SelectDefault1Case(n int) int {
	ch := make(chan int, 1)
	count := 0
	for range n {
		ch <- 1
		select {
		case v := <-ch:
			count += v
		default:
		}
	}
	return count
}

// ---------- 不带 default 的 select（走 selectgo） ----------

// SelectNoDefault1Case 1个 case，编译为直接 chanrecv（无 selectgo）
func SelectNoDefault1Case(n int) int {
	ch := make(chan int, 1)
	count := 0
	for range n {
		ch <- 1
		select { //nolint:gosimple // 故意使用单 case select 测试编译器优化
		case v := <-ch:
			count += v
		}
	}
	return count
}

// SelectNoDefault2Case 2个 case，走 selectgo 快速路径
func SelectNoDefault2Case(n int) int {
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

// SelectNoDefault4Case 4个 case，走完整 selectgo
func SelectNoDefault4Case(n int) int {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	ch3 := make(chan int, 1)
	ch4 := make(chan int, 1)
	count := 0
	for range n {
		ch1 <- 1
		select {
		case v := <-ch1:
			count += v
		case v := <-ch2:
			count += v
		case v := <-ch3:
			count += v
		case v := <-ch4:
			count += v
		}
	}
	return count
}

// SelectNoDefault8Case 8个 case，selectgo 开销更大
func SelectNoDefault8Case(n int) int {
	chs := make([]chan int, 8)
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
		case v := <-chs[4]:
			count += v
		case v := <-chs[5]:
			count += v
		case v := <-chs[6]:
			count += v
		case v := <-chs[7]:
			count += v
		}
	}
	return count
}
