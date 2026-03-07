package muridae

import "go-notes/goengineering/pprof-practise/animal"

type Muridae interface {
	animal.Animal
	Hole()
	Steal()
}
