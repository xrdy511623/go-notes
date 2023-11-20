package muridae

import "go-notes/go-principle-and-practise/pprof-practise/animal"

type Muridae interface {
	animal.Animal
	Hole()
	Steal()
}
