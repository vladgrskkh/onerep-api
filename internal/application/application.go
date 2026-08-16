package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	redislib "github.com/redis/go-redis/v9"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"

	"github.com/vladgrskkh/onerep-api/internal/config"
	exercisehandler "github.com/vladgrskkh/onerep-api/internal/handler/exercise"
	mediahandler "github.com/vladgrskkh/onerep-api/internal/handler/media"
	progresshandler "github.com/vladgrskkh/onerep-api/internal/handler/progress"
	templatehandler "github.com/vladgrskkh/onerep-api/internal/handler/template"
	workouthandler "github.com/vladgrskkh/onerep-api/internal/handler/workout"
	validatorjwt "github.com/vladgrskkh/onerep-api/internal/infrastructure/auth/jwt"
	bodyweightpostgres "github.com/vladgrskkh/onerep-api/internal/infrastructure/bodyweight/postgres"
	exercise_postgres "github.com/vladgrskkh/onerep-api/internal/infrastructure/exercise/postgres"
	medias3 "github.com/vladgrskkh/onerep-api/internal/infrastructure/media/s3"
	progresspostgres "github.com/vladgrskkh/onerep-api/internal/infrastructure/progress/postgres"
	templatepostgres "github.com/vladgrskkh/onerep-api/internal/infrastructure/template/postgres"
	workoutpostgres "github.com/vladgrskkh/onerep-api/internal/infrastructure/workout/postgres"
	workoutredis "github.com/vladgrskkh/onerep-api/internal/infrastructure/workout/redis"
	bodyweightsvc "github.com/vladgrskkh/onerep-api/internal/service/bodyweight"
	exerciseSvc "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	progresssvc "github.com/vladgrskkh/onerep-api/internal/service/progress"
	templatesvc "github.com/vladgrskkh/onerep-api/internal/service/template"
	workoutsvc "github.com/vladgrskkh/onerep-api/internal/service/workout"
)

type Application struct {
	logger      *slog.Logger
	cfg         *config.Config
	server      *http.Server
	pool        *pgxpool.Pool
	redisClient *redislib.Client
	storage     *medias3.Storage
	validator   *validatorjwt.Validator

	exerciseSvc   *exerciseSvc.ExerciseService
	templateSvc   *templatesvc.TemplateService
	workoutSvc    *workoutsvc.WorkoutService
	progressSvc   *progresssvc.ProgressService
	bodyWeightSvc *bodyweightsvc.BodyWeightService

	exerciseHandler *exercisehandler.ExerciseHandler
	templateHandler *templatehandler.TemplateHandler
	workoutHandler  *workouthandler.WorkoutHandler
	progressHandler *progresshandler.ProgressHandler
	mediaHandler    *mediahandler.MediaHandler
	volumeWorker    *workoutredis.Worker
}

type Option func(*Application) error

func WithLogger(logger *slog.Logger) Option {
	return func(a *Application) error {
		a.logger = logger
		return nil
	}
}

// WithDatabase connects to PostgreSQL using the configured URL.
func WithDatabase(ctx context.Context) Option {
	return func(a *Application) error {
		pool, err := pgxpool.New(ctx, a.cfg.DatabaseURL)
		if err != nil {
			return fmt.Errorf("connect to postgres: %w", err)
		}
		a.pool = pool
		return nil
	}
}

// WithRedis connects to Redis using the configured URL.
func WithRedis(_ context.Context) Option {
	return func(a *Application) error {
		opts, err := redislib.ParseURL(a.cfg.RedisURL)
		if err != nil {
			return fmt.Errorf("parse redis url: %w", err)
		}
		a.redisClient = redislib.NewClient(opts)
		return nil
	}
}

// WithS3 connects to the S3-compatible object store using the configured
// endpoint and credentials.
func WithS3(ctx context.Context) Option {
	return func(a *Application) error {
		storage, err := medias3.NewStorage(
			ctx,
			a.cfg.S3Endpoint,
			a.cfg.S3AccessKey,
			a.cfg.S3SecretKey,
			a.cfg.S3Bucket,
			a.cfg.S3UseSSL,
		)
		if err != nil {
			return fmt.Errorf("connect to s3: %w", err)
		}
		a.storage = storage
		return nil
	}
}

// WithJWKS builds the auth token validator and fetches the JWKS once so the
// first authenticated request has a warm cache.
func WithJWKS(ctx context.Context) Option {
	return func(a *Application) error {
		a.validator = validatorjwt.NewValidator(a.cfg.AuthBaseURL, nil)
		if err := a.validator.Refresh(ctx); err != nil {
			return fmt.Errorf("fetch jwks: %w", err)
		}
		return nil
	}
}

// WithServices builds the repositories, services, queue and worker from the
// database pool and redis client. It must follow WithDatabase and WithRedis.
func WithServices() Option {
	return func(a *Application) error {
		if a.pool == nil {
			return errors.New(
				"database pool not initialized: use application.WithDatabase before application.WithServices",
			)
		}
		if a.redisClient == nil {
			return errors.New(
				"redis client not initialized: use application.WithRedis before application.WithServices",
			)
		}
		if a.storage == nil {
			return errors.New(
				"s3 storage not initialized: use application.WithS3 before application.WithServices",
			)
		}

		trManager, err := manager.New(trmpgx.NewDefaultFactory(a.pool))
		if err != nil {
			return fmt.Errorf("create transaction manager: %w", err)
		}

		exerciseRepo := exercise_postgres.NewExerciseRepo(a.pool)
		muscleGroupRepo := exercise_postgres.NewMuscleGroupRepo(a.pool)
		templateRepo := templatepostgres.NewTemplateRepo(a.pool)
		workoutRepo := workoutpostgres.NewWorkoutRepo(a.pool)
		bodyWeightRepo := bodyweightpostgres.NewBodyWeightRepo(a.pool)
		progressRepo := progresspostgres.NewProgressRepo(a.pool)

		a.exerciseSvc = exerciseSvc.NewExerciseService(
			exerciseRepo,
			muscleGroupRepo,
			trManager,
			a.storage,
			a.cfg.MediaUploadTTL,
		)
		a.templateSvc = templatesvc.NewTemplateService(
			templateRepo,
			trManager,
			a.storage,
			a.cfg.MediaUploadTTL,
		)
		a.progressSvc = progresssvc.NewProgressService(progressRepo)
		a.bodyWeightSvc = bodyweightsvc.NewBodyWeightService(bodyWeightRepo)

		queue := workoutredis.NewQueue(a.redisClient)
		a.workoutSvc = workoutsvc.NewWorkoutService(
			workoutRepo,
			templateRepo,
			progressRepo,
			queue,
			trManager,
		)

		calculator := workoutsvc.NewVolumeCalculator(
			workoutRepo,
			exerciseRepo,
			progressRepo,
			trManager,
		)
		a.volumeWorker = workoutredis.NewWorker(queue, calculator, a.logger)

		return nil
	}
}

// WithHandlers builds the HTTP handlers and must follow WithServices.
func WithHandlers() Option {
	return func(a *Application) error {
		if a.exerciseSvc == nil || a.templateSvc == nil || a.workoutSvc == nil {
			return errors.New(
				"services not initialized: use application.WithServices before application.WithHandlers",
			)
		}

		a.exerciseHandler = exercisehandler.NewExerciseHandler(a.exerciseSvc, a.logger)
		a.templateHandler = templatehandler.NewTemplateHandler(a.templateSvc, a.logger)
		a.workoutHandler = workouthandler.NewWorkoutHandler(a.workoutSvc, a.logger)
		a.progressHandler = progresshandler.NewProgressHandler(
			a.progressSvc,
			a.bodyWeightSvc,
			a.logger,
		)
		a.mediaHandler = mediahandler.NewMediaHandler(
			a.exerciseSvc,
			a.templateSvc,
			a.logger,
		)

		return nil
	}
}

func New(opts ...Option) (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	app := &Application{
		logger: logger,
		cfg:    cfg,
	}

	for _, opt := range opts {
		if err := opt(app); err != nil {
			return nil, err
		}
	}

	return app, nil
}

func (app *Application) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	if app.pool == nil {
		return errors.New("database pool not initialized: use application.WithDatabase")
	}
	if app.redisClient == nil {
		return errors.New("redis client not initialized: use application.WithRedis")
	}
	if app.validator == nil {
		return errors.New("jwt validator not initialized: use application.WithJWKS")
	}
	if app.volumeWorker == nil {
		return errors.New("worker not initialized: use application.WithServices")
	}

	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()

	workerDone := make(chan struct{})
	go func() {
		app.volumeWorker.Run(workerCtx)
		close(workerDone)
	}()

	app.server = &http.Server{
		Addr:              ":" + app.cfg.Port,
		Handler:           app.RegisterRoutes(app.validator),
		ReadHeaderTimeout: app.cfg.ReadHeaderTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		app.logger.Info("starting gym api", "port", app.cfg.Port)
		errCh <- app.server.ListenAndServe()
	}()

	var runErr error
	select {
	case <-ctx.Done():
		app.logger.Info("shutting down...")
		shutdownCtx, shutdownCancel := context.WithTimeout(
			context.Background(),
			app.cfg.ShutdownTimeout,
		)
		defer shutdownCancel()

		if err := app.server.Shutdown(shutdownCtx); err != nil {
			runErr = err
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			runErr = err
		}
	}

	// The worker must exit before its database pool and redis client are
	// closed; cancelWorker stops it and workerDone waits for it.
	cancelWorker()
	<-workerDone

	_ = app.redisClient.Close()
	app.pool.Close()

	return runErr
}
