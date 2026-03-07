package main

import "fmt"

type StudentDetail struct {
	Name string
	Age  int
}

func RegisterStudent(name string, age int) *StudentDetail {
	s := new(StudentDetail)
	s.Name = name
	s.Age = age
	return s
}

func main() {
	student := RegisterStudent("Jim", 18)
	fmt.Println(student.Name, student.Age)
}
