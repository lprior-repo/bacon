package common

import (
	"context"
	"fmt"
)

type Pipeline[T any] struct {
	steps []func(T) (T, error)
}

func NewPipeline[T any]() *Pipeline[T] {
	return &Pipeline[T]{
		steps: make([]func(T) (T, error), 0),
	}
}

func (p *Pipeline[T]) AddStep(step func(T) (T, error)) *Pipeline[T] {
	p.steps = append(p.steps, step)
	return p
}

func (p *Pipeline[T]) Execute(input T) (T, error) {
	current := input
	for i, step := range p.steps {
		result, err := step(current)
		if err != nil {
			return current, fmt.Errorf("step %d failed: %w", i, err)
		}
		current = result
	}
	return current, nil
}

func Map[T, R any](items []T, transform func(T) R) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = transform(item)
	}
	return result
}

func Filter[T any](items []T, predicate func(T) bool) []T {
	var result []T
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

func Reduce[T, R any](items []T, initial R, reducer func(R, T) R) R {
	result := initial
	for _, item := range items {
		result = reducer(result, item)
	}
	return result
}

type Result[T any] struct {
	Value T
	Error error
}

func Ok[T any](value T) Result[T] {
	return Result[T]{Value: value, Error: nil}
}

func Err[T any](err error) Result[T] {
	var zero T
	return Result[T]{Value: zero, Error: err}
}

func (r Result[T]) IsSuccess() bool {
	return r.Error == nil
}

func (r Result[T]) IsFailure() bool {
	return r.Error != nil
}

func WithTracedPipeline[T any](ctx context.Context, name string, pipeline *Pipeline[T], input T) Result[T] {
	var result Result[T]
	
	err := WithTracing(ctx, name, func(tracedCtx context.Context) error {
		value, err := pipeline.Execute(input)
		if err != nil {
			result = Err[T](err)
			return err
		}
		result = Ok(value)
		return nil
	})
	
	if err != nil {
		return Err[T](err)
	}
	return result
}