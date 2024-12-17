package main

import "fmt"

func generateSlice() []int {
	s := []int{}
	for i := 0; i < 3; i++ {
		s = append(s, i)
	}
	return s
}

func modifyOne(s []int) {
	s[0] = 1024
	// []int{1024, 1, 2}
	fmt.Printf("s in modifyOne is:%v\n", s)
}

func modifyTwo(s []int) {
	s = append(s, 2048)
	s[0] = 1024
	// []int{1024, 1, 2, 2048}
	fmt.Printf("s in modifyTwo is:%v\n", s)
}

func modifyThree(s []int) {
	s = append(s, 2048)
	s = append(s, 4096)
	s[0] = 1024
	// []int{1024, 1, 2, 2048, 4096}
	fmt.Printf("s in modifyThree is:%v\n", s)
}

func modifyFour(s []int) {
	s[0] = 1024
	s = append(s, 2048)
	s = append(s, 4096)
	// []int{1024, 1, 2, 2048, 4096}
	fmt.Printf("s in modifyFour is:%v\n", s)
}

func modifyFive(s []int) {
	s1 := append(s, 2048)
	// []int{0, 1, 2, 2048}
	fmt.Println(s1)
	s2 := append(s, 4096)
	// []int{0, 1, 2, 4096}, []int{0, 1, 2, 4096}, []int{0, 1, 2}
	fmt.Printf("in modifyFive s1 is:%v; s2 is:%v; s is: %v\n", s1, s2, s)
}

func SliceRiseOne(s []int) {
	s = append(s, 0)
	for i := range s {
		s[i]++
	}
}

func SliceRiseTwo(s []int) {
	s = append(s, 0)
	for _, v := range s {
		v++
	}
}

// 	VerifySliceExpand 切片扩容策略
func VerifySliceExpand() {
	s := make([]int, 0)
	oldCap := cap(s)
	for i := 0; i < 2048; i++ {
		s = append(s, i)
		newCap := cap(s)
		if newCap != oldCap {
			fmt.Printf("[%d -> %4d] cap = %-4d  |  after append %-4d  cap = %-4d\n", 0, i-1, oldCap, i, newCap)
			oldCap = newCap
		}
	}
}

func main() {
	s1 := generateSlice()
	modifyOne(s1)
	// []int{1024, 1, 2}
	fmt.Println(s1)

	s2 := generateSlice()
	modifyTwo(s2)
	// []int{1024, 1, 2}
	fmt.Println(s2)

	s3 := generateSlice()
	modifyThree(s3)
	// []int{0, 1, 2}
	fmt.Println(s3)

	s4 := generateSlice()
	modifyFour(s4)
	// []int{1024, 1, 2}
	fmt.Println(s4)

	s5 := generateSlice()
	modifyFive(s5)
	// []int{0, 1, 2}
	fmt.Println(s5)

	s6 := []int{1, 2}
	s7 := s6
	s7 = append(s7, 3)
	SliceRiseOne(s6)
	SliceRiseOne(s7)
	// []int{1, 2} []int{2, 3, 4}
	fmt.Println(s6, s7)

	SliceRiseTwo(s6)
	SliceRiseTwo(s7)
	// []int{1, 2} []int{2, 3, 4}
	fmt.Println(s6, s7)

	VerifySliceExpand()
}
