package felidae

import "go-notes/go-principle-and-practise/pprof-practise/animal"

type Felidae interface {
	animal.Animal
	Climb()
	Sneak()
}
