package main

import (
	"fmt"
	"math/rand"
	"time"
)

func abs(x int) float32 {
	if x < 0 {
		return float32(-x)
	}
	return float32(x)
}

type Params struct {
	blocks        []int
	classes       []int
	hours         []int
	teachers      []int
	popSize       int
	crossoverProb float32
	mutateProb    float32
	restrictions  *Restrictions
}

type Teacher struct {
	hours   []int
	classes []int
}

type TeacherHours struct {
	min int
	max int
}

type Restrictions struct {
	classes              []int
	teachers             []TeacherHours
	classrooms           []int
	classPonderation     float32
	teacherPonderation   float32
	classroomPonderation float32
}

type Individual struct {
	solutions  []int
	classes    []int
	teachers   []int
	classrooms []int
	fitness    float32
}

type Population []*Individual

//***********************************************************************************+

func Mutate(ind *Individual, params *Params) {

	b1 := rand.Intn(len(params.blocks) - 1)
	if ind.solutions[b1] != -1 {
		idx := params.blocks[b1] + ind.solutions[b1]
		ind.fitness += ReverseFitness(ind, params, idx)
		ind.classes[params.classes[idx]]--
		ind.classrooms[params.hours[idx]]--
		ind.teachers[params.teachers[idx]]--
		ind.fitness -= ReverseFitness(ind, params, idx)
	}
	ind.solutions[b1] = rand.Intn(params.blocks[b1+1]-params.blocks[b1]+1) - 1
	if ind.solutions[b1] != -1 {
		idx := params.blocks[b1] + ind.solutions[b1]
		ind.fitness += ReverseFitness(ind, params, idx)
		ind.classes[params.classes[idx]]++
		ind.classrooms[params.hours[idx]]++
		ind.teachers[params.teachers[idx]]++
		ind.fitness -= ReverseFitness(ind, params, idx)
	}
}

func ReverseFitness(ind *Individual, params *Params, i int) float32 {

	var fitness float32 = 0.0

	j := params.classes[i]
	fitness += abs(ind.classes[j]-params.restrictions.classes[j]) * params.restrictions.classPonderation
	j = params.hours[i]
	if ind.classrooms[j] > params.restrictions.classrooms[j] {
		fitness += abs(ind.classrooms[j]-params.restrictions.classrooms[j]) * params.restrictions.classroomPonderation
	}
	j = params.teachers[i]
	if ind.teachers[j] < params.restrictions.teachers[j].min {
		fitness += abs(ind.teachers[j]-params.restrictions.teachers[j].min) * params.restrictions.teacherPonderation
	} else if ind.teachers[j] > params.restrictions.teachers[j].max {
		fitness += abs(ind.teachers[j]-params.restrictions.teachers[j].max) * params.restrictions.teacherPonderation
	}
	return fitness
}

func ReverseTwoFitness(ind1 *Individual, ind2 *Individual, params *Params, i int) {
	ind1.fitness += ReverseFitness(ind1, params, i)
	ind1.classes[params.classes[i]]--
	ind1.teachers[params.teachers[i]]--
	ind1.classrooms[params.hours[i]]--
	ind1.fitness -= ReverseFitness(ind1, params, i)

	ind2.fitness += ReverseFitness(ind2, params, i)
	ind2.classes[params.classes[i]]++
	ind2.teachers[params.teachers[i]]++
	ind2.classrooms[params.hours[i]]++
	ind2.fitness -= ReverseFitness(ind2, params, i)
}

func Crossover(ind1 *Individual, ind2 *Individual, params *Params) {

	b1 := rand.Intn(len(params.blocks) - 1)

	if ind1.solutions[b1] == ind2.solutions[b1] {
		return
	}

	if ind1.solutions[b1] != -1 {
		ReverseTwoFitness(ind1, ind2, params, params.blocks[b1]+ind1.solutions[b1])
	}
	if ind2.solutions[b1] != -1 {
		ReverseTwoFitness(ind2, ind1, params, params.blocks[b1]+ind2.solutions[b1])
	}

	ind1.solutions[b1], ind2.solutions[b1] = ind2.solutions[b1], ind1.solutions[b1]

}

func ComputeNewFitness(individual *Individual, params *Params) float32 {

	individual.classes = make([]int, len(params.restrictions.classes))
	individual.teachers = make([]int, len(params.restrictions.teachers))
	individual.classrooms = make([]int, len(params.restrictions.classrooms))

	for i := range individual.solutions {
		if individual.solutions[i] != -1 {
			individual.classes[params.classes[params.blocks[i]+individual.solutions[i]]]++
			individual.teachers[params.teachers[params.blocks[i]+individual.solutions[i]]]++
			individual.classrooms[params.hours[params.blocks[i]+individual.solutions[i]]]++
		}
	}

	var fitness float32 = 0.0
	for i := range individual.classes {
		fitness -= abs(individual.classes[i]-params.restrictions.classes[i]) * params.restrictions.classPonderation
	}

	for i := range individual.teachers {
		if individual.teachers[i] < params.restrictions.teachers[i].min {
			fitness -= abs(individual.teachers[i]-params.restrictions.teachers[i].min) * params.restrictions.teacherPonderation
		} else if individual.teachers[i] > params.restrictions.teachers[i].max {
			fitness -= abs(individual.teachers[i]-params.restrictions.teachers[i].max) * params.restrictions.teacherPonderation
		}
	}

	for i := range individual.classrooms {
		if individual.classrooms[i] > params.restrictions.classrooms[i] {
			fitness -= abs(individual.classrooms[i]-params.restrictions.classrooms[i]) * params.restrictions.classroomPonderation
		}

	}

	return fitness

}

func rCreate(individual *Individual, params *Params) {
	individual.solutions = make([]int, len(params.blocks)-1)
	for i := range individual.solutions {
		individual.solutions[i] = rand.Intn(params.blocks[i+1]-params.blocks[i]+1) - 1
	}
	individual.fitness = ComputeNewFitness(individual, params)
}

func bestIndividual(p Population, params *Params) *Individual {
	best := p[0]
	for i := 0; i < params.popSize; i++ {
		if p[i].fitness > best.fitness {
			best = p[i]
		}
	}
	return best

}

//***********************************************************************************+

func gCrossover(p Population, params *Params) {
	c := make(chan struct{})
	for i := 0; i < params.popSize; i += 2 {
		go func(i int, c chan struct{}) {
			if rand.Float32() < params.crossoverProb {
				Crossover(p[i], p[i+1], params)
			}
			c <- struct{}{}
		}(i, c)
	}
	for i := 0; i < params.popSize; i += 2 {
		<-c
	}
}

func Select(p Population, c chan *Individual, params *Params) {
	r := rand.Intn(params.popSize)

	best := p[rand.Intn(params.popSize)]

	if best.fitness < p[r].fitness {
		best = p[r]
	}

	c <- best

}

func TournamentSelect(p Population, params *Params) Population {

	c := make(chan *Individual, params.popSize)

	for i := 0; i < params.popSize; i++ {
		go Select(p, c, params)
	}

	var foo Population

	for i := 0; i < params.popSize; i++ {
		foo = append(foo, <-c)
	}

	return foo

}

func solveConcurrent(params *Params) (*Individual, float32) {

	population := make(Population, params.popSize)

	for i := range population {
		population[i] = &Individual{}
		rCreate(population[i], params)
	}

	generation := 0

	var nMutations float32 = 0

	for bestIndividual(population, params).fitness != 0 && generation < 1000 {
		generation++
		population = TournamentSelect(population, params)
		gCrossover(population, params)
		nMutations += float32(len(params.blocks)) * float32(params.popSize) * params.mutateProb
		for nMutations >= 1 {
			Mutate(population[rand.Intn(params.popSize)], params)
			nMutations--
		}
	}

	return bestIndividual(population, params), float32(generation)
}

//****************************************************************************************

func computeParams(teachers []Teacher, restrictions *Restrictions) *Params {

	params := &Params{}

	params.restrictions = restrictions
	params.popSize = 150
	params.crossoverProb = 0.8
	params.mutateProb = 0.05

	size := 0
	help := make([]int, len(teachers))
	for i := range teachers {
		size += len(teachers[i].classes) * len(teachers[i].hours)
		help[i] = len(teachers[i].classes) * len(teachers[i].hours)
	}

	params.teachers = make([]int, size)
	params.classes = make([]int, 0)
	params.blocks = make([]int, 0)
	params.blocks = append(params.blocks, 0)
	aux := 0
	aux2 := 0
	for i, x := range help {
		aux2 = aux + x
		for j := aux; j < aux2; j++ {
			params.teachers[j] = i
		}
		aux = aux2
	}

	aux = 0
	aux3 := 0
	for i := range teachers {
		aux = help[i] / len(teachers[i].classes)
		for j := 0; j < aux; j++ {
			params.classes = append(params.classes, teachers[i].classes...)
		}
		aux2 = help[i] / len(teachers[i].hours)
		for j := 0; j < aux; j++ {
			for k := 0; k < aux2; k++ {
				params.hours = append(params.hours, teachers[i].hours[j])
			}
		}

		for j := 0; j < len(teachers[i].hours); j++ {
			aux3 += len(teachers[i].classes)
			params.blocks = append(params.blocks, aux3)
		}
	}

	return params

}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	teachers := make([]Teacher, 5)

	teachers[0].hours = []int{1, 2}
	teachers[1].hours = []int{0, 1, 2, 3, 4}
	teachers[2].hours = []int{1, 2, 3}
	teachers[3].hours = []int{0, 1, 2, 4}
	teachers[4].hours = []int{0, 2, 3}

	teachers[0].classes = []int{0, 1}
	teachers[1].classes = []int{1, 2, 3}
	teachers[2].classes = []int{0, 2}
	teachers[3].classes = []int{0, 3}
	teachers[4].classes = []int{0, 1, 2}

	restrictions := Restrictions{}

	restrictions.classes = make([]int, 4)
	restrictions.teachers = make([]TeacherHours, 5)
	restrictions.classrooms = make([]int, 5)

	restrictions.classrooms[0] = 4
	restrictions.classrooms[1] = 4
	restrictions.classrooms[2] = 4
	restrictions.classrooms[3] = 4
	restrictions.classrooms[4] = 4

	restrictions.classPonderation = 1
	restrictions.teacherPonderation = 1
	restrictions.classroomPonderation = 1

	restrictions.classes[0] = 2
	restrictions.classes[1] = 2
	restrictions.classes[2] = 3
	restrictions.classes[3] = 2

	restrictions.teachers[0].min = 1
	restrictions.teachers[0].max = 2
	restrictions.teachers[1].min = 3
	restrictions.teachers[1].max = 4
	restrictions.teachers[2].min = 1
	restrictions.teachers[2].max = 2
	restrictions.teachers[3].min = 1
	restrictions.teachers[3].max = 2
	restrictions.teachers[4].min = 2
	restrictions.teachers[4].max = 4

	params := computeParams(teachers, &restrictions)

	soluciones := 0
	var generation float32 = 0
	for i := 0; i < 1000; i++ {
		ind, gen := solveConcurrent(params)
		if ind.fitness == 0 {
			soluciones++
			generation += gen
		}
	}

	fmt.Println("Soluciones % ", soluciones)
	fmt.Println("En aprox ", generation/1000.0, " generaciones")
}
