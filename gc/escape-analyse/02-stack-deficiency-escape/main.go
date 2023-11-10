package main

func MakeSlice() {
	s := make([]int, 10000, 10000)
	for i := range s {
		s[i] = i
	}
}

func main() {
	MakeSlice()
}
