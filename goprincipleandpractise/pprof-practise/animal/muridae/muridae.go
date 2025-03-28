package muridae

import "go-notes/goprincipleandpractise/pprof-practise/animal"

type Muridae interface {
	animal.Animal
	Hole()
	Steal()
}
