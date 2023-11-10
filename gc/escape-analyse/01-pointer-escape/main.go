package main

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
	RegisterStudent("Jim", 18)
}
