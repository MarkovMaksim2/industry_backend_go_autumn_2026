package main

import (
	"errors"
	"fmt"
)

var ErrZero = errors.New("zero is not allowed")

func fizzBuzz(n int) (string, error) {
	if n == 0 {
		return "", ErrZero
	}

	s := make([]rune, 0, 4)

	if n%3 == 0 {
		s = append(s, []rune("Fizz")...)
	}

	if n%5 == 0 {
		s = append(s, []rune("Buzz")...)
	}

	if len(s) == 0 {
		return fmt.Sprintf("%d", n), nil
	}

	return string(s), nil
}
