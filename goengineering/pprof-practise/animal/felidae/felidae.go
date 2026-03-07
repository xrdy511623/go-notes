package felidae

import "go-notes/goengineering/pprof-practise/animal"

type Felidae interface {
	animal.Animal
	Climb()
	Sneak()
}
