package main

import "fmt"

// StructA 包含指针 (string 内部有指针)
type StructA struct {
	ID   int
	Name string // 指针类型
	Data [10]int
}

// StructB 不含指针
type StructB struct {
	ID   int
	Flag bool
	Data [10]int // 值类型数组
}

// 使用索引代替指针

type Node struct{ value int }
type Graph struct {
	nodes []Node
	edges [][]int // 存储索引，而非 *Node
}

func main() {
	a := StructA{ID: 1, Name: "contains pointer"}
	b := StructB{ID: 2, Flag: true}
	// GC扫描 a 时需要检查 Name 字段指向的字符串数据
	// GC扫描 b 时，检查完 Flag 就可以跳过 Data 数组内部（因为它不含指针）
	fmt.Println(a.Name, b.Flag)

	// 对于 Graph，GC 只需扫描 nodes 切片头和 edges 切片头及其内部的 int 切片头
	// 无需深入扫描 Node 结构体内部或跟踪大量节点间指针
}
