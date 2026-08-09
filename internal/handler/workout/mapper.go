package workout

import (
	"github.com/google/uuid"

	domainworkout "github.com/vladgrskkh/onerep-api/internal/domain/workout"
	"github.com/vladgrskkh/onerep-api/internal/handler/workout/dto"
	serviceworkout "github.com/vladgrskkh/onerep-api/internal/service/workout"
)

// toStartCommand maps the HTTP request to the service start command.
func toStartCommand(req dto.StartWorkoutRequest, userID uuid.UUID) serviceworkout.StartWorkoutCommand {
	return serviceworkout.StartWorkoutCommand{UserID: userID, TemplateID: req.TemplateID}
}

// toAddExerciseCommand maps the HTTP request to the service add-exercise
// command.
func toAddExerciseCommand(req dto.AddExerciseRequest, workoutID uuid.UUID) serviceworkout.AddExerciseCommand {
	return serviceworkout.AddExerciseCommand{WorkoutID: workoutID, ExerciseID: req.ExerciseID}
}

// toLogSetCommand maps the HTTP request to the service log-set command.
func toLogSetCommand(req dto.LogSetRequest, workoutID, workoutExerciseID uuid.UUID) serviceworkout.LogSetCommand {
	return serviceworkout.LogSetCommand{
		WorkoutID:         workoutID,
		WorkoutExerciseID: workoutExerciseID,
		WeightKg:          req.WeightKg,
		Reps:              req.Reps,
		RPE:               req.RPE,
		RestSeconds:       req.RestSeconds,
		IsWarmup:          req.IsWarmup,
	}
}

// toFinishCommand maps the path parameter to the service finish command.
func toFinishCommand(workoutID uuid.UUID) serviceworkout.FinishWorkoutCommand {
	return serviceworkout.FinishWorkoutCommand{WorkoutID: workoutID}
}

// toWorkoutResponse maps the domain workout to the HTTP response.
func toWorkoutResponse(w domainworkout.Workout) dto.WorkoutResponse {
	resp := dto.WorkoutResponse{
		ID:         w.ID,
		UserID:     w.UserID,
		TemplateID: w.TemplateID,
		StartedAt:  w.StartedAt,
		FinishedAt: w.FinishedAt,
		Notes:      w.Notes,
		CreatedAt:  w.CreatedAt,
		UpdatedAt:  w.UpdatedAt,
	}
	for _, e := range w.Exercises {
		resp.Exercises = append(resp.Exercises, toWorkoutExerciseResponse(e))
	}
	return resp
}

// toWorkoutListResponse maps a list of domain workouts to HTTP responses.
func toWorkoutListResponse(workouts []domainworkout.Workout) []dto.WorkoutResponse {
	resp := make([]dto.WorkoutResponse, 0, len(workouts))
	for _, w := range workouts {
		resp = append(resp, toWorkoutResponse(w))
	}
	return resp
}

// toWorkoutExerciseResponse maps the domain workout exercise to the HTTP
// response.
func toWorkoutExerciseResponse(e domainworkout.WorkoutExercise) dto.WorkoutExerciseResponse {
	resp := dto.WorkoutExerciseResponse{
		ID:         e.ID,
		ExerciseID: e.ExerciseID,
		SortOrder:  e.SortOrder,
		Notes:      e.Notes,
	}
	for _, s := range e.Sets {
		resp.Sets = append(resp.Sets, toWorkoutSetResponse(s))
	}
	return resp
}

// toWorkoutSetResponse maps the domain workout set to the HTTP response.
func toWorkoutSetResponse(s domainworkout.WorkoutSet) dto.WorkoutSetResponse {
	return dto.WorkoutSetResponse{
		ID:          s.ID,
		SetNumber:   s.SetNumber,
		WeightKg:    s.WeightKg,
		Reps:        s.Reps,
		RPE:         s.RPE,
		RestSeconds: s.RestSeconds,
		IsWarmup:    s.IsWarmup,
	}
}

// toLogSetResponse maps the service set result to the HTTP response.
func toLogSetResponse(result serviceworkout.SetResult) dto.LogSetResponse {
	return dto.LogSetResponse{
		WorkoutSetResponse: toWorkoutSetResponse(result.WorkoutSet),
		IsPR:               result.IsPR,
		Estimated1RM:       result.Estimated1RM,
	}
}
