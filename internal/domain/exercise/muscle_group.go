package exercise

import "strings"

type MuscleGroup struct {
	ID   int
	Name string
}

func NewMuscleGroup(id int, name string) (MuscleGroup, error) {
	if id <= 0 {
		return MuscleGroup{}, ErrInvalidID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return MuscleGroup{}, ErrInvalidName
	}
	return MuscleGroup{
		ID:   id,
		Name: name,
	}, nil
}
