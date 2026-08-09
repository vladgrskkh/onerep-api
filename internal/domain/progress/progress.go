package progress

import (
	"time"

	"github.com/google/uuid"
)

type Progress1RM struct {
	ExerciseID   uuid.UUID
	UserID       uuid.UUID
	Date         time.Time
	Estimated1RM float64
}

type ProgressVolume struct {
	MuscleGroupID int
	UserID        uuid.UUID
	Date          time.Time
	TotalKG       float64
}

func NewProgress1RM(exerciseID, userID uuid.UUID, date time.Time, estimated1RM float64) (Progress1RM, error) {
	if exerciseID == uuid.Nil {
		return Progress1RM{}, ErrInvalidExerciseID
	}
	if userID == uuid.Nil {
		return Progress1RM{}, ErrInvalidUserID
	}
	if estimated1RM <= 0 {
		return Progress1RM{}, ErrInvalidEstimated1RM
	}
	return Progress1RM{
		ExerciseID:   exerciseID,
		UserID:       userID,
		Date:         date,
		Estimated1RM: estimated1RM,
	}, nil
}

func NewProgressVolume(muscleGroupID int, userID uuid.UUID, date time.Time, totalKG float64) (ProgressVolume, error) {
	if userID == uuid.Nil {
		return ProgressVolume{}, ErrInvalidUserID
	}
	if muscleGroupID <= 0 {
		return ProgressVolume{}, ErrInvalidMuscleGroupID
	}
	if totalKG <= 0 {
		return ProgressVolume{}, ErrInvalidTotalKG
	}
	return ProgressVolume{
		MuscleGroupID: muscleGroupID,
		UserID:        userID,
		Date:          date,
		TotalKG:       totalKG,
	}, nil
}
