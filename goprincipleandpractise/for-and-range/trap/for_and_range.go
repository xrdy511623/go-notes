package main

import "fmt"

type person struct {
	name string
	age  int
}

func main() {
	persons := []person{{name: "John", age: 17}, {name: "Tom", age: 18}, {name: "Jack", age: 19}}
	for _, s := range persons {
		s.age += 10
	}
	fmt.Printf("persons are: %+v\n", persons)
	for i, n := 0, len(persons); i < n; i++ {
		persons[i].age += 20
	}
	/*
		persons 是一个长度为 3 的切片，每个元素是一个结构体。
		使用 range 迭代时，试图将每个结构体的 age 字段增加 10，但修改无效，因为 range 返回的是拷贝。
		使用 for 迭代时，将每个(元素)结构体的 age 字段增加 20，修改有效。
		结论：range 迭代时，返回的是拷贝。
	*/
	// [{Name:John Age:37} {Name:Tom Age:38} {Name:Jack Age:39}]
	fmt.Printf("persons are: %+v\n", persons)

	s := []int{1, 2, 3}
	/*
		这一段代码会造成死循环吗？答案：当然不会，range会对切片做拷贝，新增的数据并不在拷贝内容中，并不会发生死循环。
		for range循环其实是golang的语法糖，在循环开始前会获取切片的长度len(s), 然后再执行len(s)次数的循环
	*/
	for i := range s {
		s = append(s, i)
	}
	// []int{1,2,3,0,1,2}
	fmt.Println(s)

	n := make([]*person, 0, len(persons))
	m := make(map[string]*person)
	for _, v := range persons {
		n = append(n, &v)
		m[v.name] = &v
	}
	/*
		打印出三个相同的内存地址，也就是&{Jack 39}对应的内存地址，为什么？
		因为for range循环中，变量v是用来保存迭代切片所得的值的，因为v只被声明了一次，每次迭代的值都是赋给v，
		该变量的内存地址始终未变，这样将它的内存地址追加到新切片n中，该切片保存的都是同一个内存地址，这肯定不是我们
		预期的效果。还需要注意的是，变量v的地址也并不是指向原切片persons[2]的，因为在使用range迭代的时候，变量v
		的数据是切片的拷贝数据，所以是直接copy了结构体数据。
	*/
	for _, v := range n {
		fmt.Println(v)
	}
	for k, v := range m {
		fmt.Println(k, v)
	}
	fmt.Println("--------------------------------")
	// 可以这么改
	n = make([]*person, 0, len(persons))
	m = make(map[string]*person)
	for i, v := range persons {
		n = append(n, &persons[i])
		m[v.name] = &persons[i]
	}

	// 此时就是我们预期的效果了
	for _, v := range n {
		fmt.Println(v)
	}
	for k, v := range m {
		fmt.Println(k, v)
	}

	for k, v := range m {
		if k == "jack" {
			v.age += 1
		}
	}

	for k := range m {
		if k == "Jack" {
			m[k].age += 10
		}
	}
	// [{Name:John Age:37} {Name:Tom Age:38} {Name:Jack Age:49}]
	for k, v := range m {
		fmt.Println(k, v)
	}
}
