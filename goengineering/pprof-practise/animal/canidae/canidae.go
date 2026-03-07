package canidae

import "go-notes/goprincipleandpractise/pprof-practise/animal"

type Canidae interface {
	animal.Animal
	Run()
	Howl()
}
