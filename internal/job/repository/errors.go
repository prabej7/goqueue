package repository

import "errors"

var (
	ErrJobNotFound     = errors.New("job not found")
	ErrNoJobsAvailable = errors.New("no jobs available")
)
