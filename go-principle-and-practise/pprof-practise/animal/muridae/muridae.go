package muridae

import "go-notes/pprof-practise/animal"

type Muridae interface {
	animal.Animal
	Hole()
	Steal()
}
