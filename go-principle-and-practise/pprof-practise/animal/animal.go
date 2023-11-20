package animal

import (
	"go-notes/go-principle-and-practise/pprof-practise/animal/canidae/dog"
	"go-notes/go-principle-and-practise/pprof-practise/animal/canidae/wolf"
	"go-notes/go-principle-and-practise/pprof-practise/animal/felidae/cat"
	"go-notes/go-principle-and-practise/pprof-practise/animal/felidae/tiger"
	"go-notes/go-principle-and-practise/pprof-practise/animal/muridae/mouse"
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
