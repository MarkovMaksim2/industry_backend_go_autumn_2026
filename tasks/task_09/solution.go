package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	workers = min(workers, len(in))

	res := make([]Result[R], len(in))

	type element struct {
		index int
		value T
	}
	workerChan := make(chan element, workers)

	go func() {
		defer close(workerChan)
		for idx := range in {
			if ctx.Err() != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case workerChan <- element{index: idx, value: in[idx]}:
			}
		}

	}()

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			for val := range workerChan {
				if ctx.Err() != nil {
					return
				}
				curRes, errf := fn(ctx, val.value)

				var zero R

				mu.Lock()
				if errf != nil {
					res[val.index] = Result[R]{Value: zero, Err: errf}
				} else {
					res[val.index] = Result[R]{Value: curRes, Err: errf}
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
