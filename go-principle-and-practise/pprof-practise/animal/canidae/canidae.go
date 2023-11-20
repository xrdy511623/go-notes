package canidae

import "go-notes/go-principle-and-practise/pprof-practise/animal"

type Canidae interface {
	animal.Animal
	Run()
	Howl()
}
