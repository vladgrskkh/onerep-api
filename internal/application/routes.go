package application

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/vladgrskkh/onerep-api/docs" // swagger docs
	"github.com/vladgrskkh/onerep-api/internal/handler"
	"github.com/vladgrskkh/onerep-api/internal/handler/middleware"
	validatorjwt "github.com/vladgrskkh/onerep-api/internal/infrastructure/auth/jwt"
)

// RegisterRoutes returns the chi router with all public and authenticated
// routes. The token validator guards the authenticated group.
func (app *Application) RegisterRoutes(v *validatorjwt.Validator) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logging(app.logger))
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	r.Get("/v1/health", app.healthHandler)
	r.Get("/v1/docs/*", httpSwagger.WrapHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(v, app.logger))

		r.Get("/v1/exercises", app.exerciseHandler.List)
		r.Post("/v1/exercises", app.exerciseHandler.Create)
		r.Get("/v1/exercises/{id}", app.exerciseHandler.Get)
		r.Patch("/v1/exercises/{id}", app.exerciseHandler.Update)
		r.Delete("/v1/exercises/{id}", app.exerciseHandler.SoftDelete)
		r.Post("/v1/exercises/{id}/media", app.mediaHandler.UploadExerciseMedia)

		r.Get("/v1/templates", app.templateHandler.List)
		r.Post("/v1/templates", app.templateHandler.Create)
		r.Get("/v1/templates/{id}", app.templateHandler.Get)
		r.Patch("/v1/templates/{id}", app.templateHandler.Update)
		r.Post("/v1/templates/{id}/publish", app.templateHandler.Publish)
		r.Post("/v1/templates/{id}/fork", app.templateHandler.Fork)
		r.Delete("/v1/templates/{id}", app.templateHandler.SoftDelete)
		r.Post("/v1/templates/{id}/media", app.mediaHandler.UploadTemplateMedia)

		r.Post("/v1/workouts", app.workoutHandler.Start)
		r.Get("/v1/workouts", app.workoutHandler.List)
		r.Get("/v1/workouts/{id}", app.workoutHandler.Get)
		r.Post("/v1/workouts/{id}/exercises", app.workoutHandler.AddExercise)
		r.Post("/v1/workouts/{id}/exercises/{exId}/sets", app.workoutHandler.LogSet)
		r.Patch("/v1/workouts/{id}/finish", app.workoutHandler.Finish)

		r.Get("/v1/progress/1rm", app.progressHandler.Get1RM)
		r.Get("/v1/progress/volume", app.progressHandler.GetVolume)
		r.Get("/v1/progress/body-weight", app.progressHandler.GetBodyWeight)
		r.Post("/v1/progress/body-weight", app.progressHandler.LogBodyWeight)
	})

	return r
}

func (app *Application) healthHandler(w http.ResponseWriter, _ *http.Request) {
	handler.WriteJSON(w, app.logger, http.StatusOK, map[string]string{"status": "ok"})
}
