package animal

import (
	"go-notes/goprincipleandpractise/pprof-practise/animal/canidae/dog"
	"go-notes/goprincipleandpractise/pprof-practise/animal/canidae/wolf"
	"go-notes/goprincipleandpractise/pprof-practise/animal/felidae/cat"
	"go-notes/goprincipleandpractise/pprof-practise/animal/felidae/tiger"
	"go-notes/goprincipleandpractise/pprof-practise/animal/muridae/mouse"
)

var (
	AllAnimals = []Animal{
		&dog.Dog{},
		&wolf.Wolf{},

		&cat.Cat{},
		&tiger.Tiger{},

		&mouse.Mouse{},
	}
)

type Animal interface {
	Name() string
	Live()

	Eat()
	Drink()
	Shit()
	Pee()
}
