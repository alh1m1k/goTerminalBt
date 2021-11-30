package main

import (
	"errors"
	"math"
	"math/rand"
)

var TimeFunc404Error  = errors.New("time function does not exist")

type timeFunction func (timeFraction float64) float64

var LinearTimeFunction timeFunction = func(timeFraction float64) float64 {
	return timeFraction
}

var QuadTimeFunction timeFunction = func(timeFraction float64) float64 {
	return math.Pow(timeFraction, 2)
}

var CircTimeFunction timeFunction = func(timeFraction float64) float64 {
	return 1 - math.Sin(math.Acos(timeFraction))
}

var ElasticTimeFunction timeFunction = func(timeFraction float64) float64 {
	return math.Pow(2, 10 * (timeFraction - 1)) * math.Cos(20 * math.Pi * 1.5 / 3 * timeFraction)
}

/**
*	fading = 1 is uniform
	fading = 10 is elastic
	x = frequency
 */
var ElasticTimeFuncGenerator = func(fading float64, x float64) timeFunction {
	return func(timeFraction float64) float64 {
		return math.Pow(2, fading * (timeFraction - 1)) * math.Cos(20 * math.Pi * x / 3 * timeFraction)
	}
}

func GetTimeFunc(name string) (timeFunction, error)  {
	switch name {
	case "linear":
		return LinearTimeFunction, nil
	case "quad":
		return QuadTimeFunction, nil
	case "circ":
		return CircTimeFunction, nil
	case "elastic":
		return ElasticTimeFunction, nil
	case "random":
		return GetRandomTimeFunc(), nil
	default:
		return nil, TimeFunc404Error
	}
}

var timeFuncNames = []string{"linear", "quad", "circ", "elastic"}
func GetRandomTimeFunc() timeFunction  {
	fn, _ := GetTimeFunc(timeFuncNames[rand.Intn(len(timeFuncNames))])
	return fn
}
