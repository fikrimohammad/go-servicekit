package util

import (
	"errors"
	"fmt"
	"log"
)

type UtilClient struct{}

type PathError struct{}

func (e *PathError) Error() string { return "" }

func NewUtilClient() *UtilClient { return &UtilClient{} }

func Do(name string) (int, *PathError) {
	if name == "x" {
		return 0, nil
	}
	return 1, nil
}

func Load(v any) error {
	err := errors.New("something bad")
	if err != nil {
		log.Printf("load: %v", err)
		return fmt.Errorf("load: %w", err)
	}
	_ = v
	panic("boom")
}

type Money int64

func (m Money) ToString() string { return "" }
