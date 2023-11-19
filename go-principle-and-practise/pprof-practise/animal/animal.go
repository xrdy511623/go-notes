package animal

import (
	"go-notes/pprof-practise/animal/canidae/dog"
	"go-notes/pprof-practise/animal/canidae/wolf"
	"go-notes/pprof-practise/animal/felidae/cat"
	"go-notes/pprof-practise/animal/felidae/tiger"
	"go-notes/pprof-practise/animal/muridae/mouse"
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
