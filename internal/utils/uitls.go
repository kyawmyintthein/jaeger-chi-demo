package utils

import (
	"math/rand"
)

const (
	min = 1
	max = 50
)

func GetRandomNumber() int {
	return rand.Intn(max-min) + min
}
