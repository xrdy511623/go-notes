package canidae

import "go-notes/goengineering/pprof-practise/animal"

type Canidae interface {
	animal.Animal
	Run()
	Howl()
}
