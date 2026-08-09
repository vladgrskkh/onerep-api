package exercise

type MuscleGroup struct {
	ID   int
	Name string
}

func NewMuscleGroup(id int, name string) MuscleGroup {
	return MuscleGroup{
		ID:   id,
		Name: name,
	}
}
