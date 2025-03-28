package felidae

import "go-notes/goprincipleandpractise/pprof-practise/animal"

type Felidae interface {
	animal.Animal
	Climb()
	Sneak()
}
