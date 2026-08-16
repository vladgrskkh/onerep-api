package exercise_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domainexercise "github.com/vladgrskkh/onerep-api/internal/domain/exercise"
	serviceexercise "github.com/vladgrskkh/onerep-api/internal/service/exercise"
	exercisemocks "github.com/vladgrskkh/onerep-api/internal/service/exercise/mocks"
)

// uploadTTL is the presigned upload URL lifetime used across the tests.
const uploadTTL = 15 * time.Minute

type ServiceTestSuite struct {
	suite.Suite

	svc             *serviceexercise.ExerciseService
	exerciseRepo    *exercisemocks.MockExerciseRepository
	muscleGroupRepo *exercisemocks.MockMuscleGroupRepository
	trManager       *exercisemocks.MockTransactionManager
	storage         *exercisemocks.MockObjectStorage
}

func (s *ServiceTestSuite) SetupTest() {
	s.exerciseRepo = exercisemocks.NewMockExerciseRepository(s.T())
	s.muscleGroupRepo = exercisemocks.NewMockMuscleGroupRepository(s.T())
	s.trManager = exercisemocks.NewMockTransactionManager(s.T())
	s.storage = exercisemocks.NewMockObjectStorage(s.T())
	s.svc = serviceexercise.NewExerciseService(
		s.exerciseRepo,
		s.muscleGroupRepo,
		s.trManager,
		s.storage,
		uploadTTL,
	)
}

// expectTx runs the transaction closure inline.
func (s *ServiceTestSuite) expectTx() {
	s.trManager.EXPECT().
		Do(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		RunAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})
}

func (s *ServiceTestSuite) TestList_PassesFilters() {
	since := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	filter := domainexercise.ExerciseFilter{
		Search:      "squat",
		MuscleGroup: "legs",
		Since:       since,
	}
	expected := []*domainexercise.Exercise{{ID: uuid.Must(uuid.NewV7()), Name: "Squat"}}
	s.exerciseRepo.EXPECT().List(mock.Anything, filter).Return(expected, nil)

	got, err := s.svc.List(context.Background(), serviceexercise.ListExercisesCommand{
		Search:      "squat",
		MuscleGroup: "legs",
		Since:       since,
	})
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(expected[0], got[0])
}

func (s *ServiceTestSuite) TestGet_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	expected := &domainexercise.Exercise{ID: exerciseID, Name: "Bench Press"}
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).Return(expected, nil)

	got, err := s.svc.Get(context.Background(), exerciseID)
	s.Require().NoError(err)
	s.Equal(expected, got)
}

func (s *ServiceTestSuite) TestGet_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(nil, domainexercise.ErrExerciseNotFound)

	_, err := s.svc.Get(context.Background(), exerciseID)
	s.Require().ErrorIs(err, domainexercise.ErrExerciseNotFound)
}

func (s *ServiceTestSuite) TestCreate_Success() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	groups := []domainexercise.ExerciseMuscleGroup{{MuscleGroupID: 1}, {MuscleGroupID: 2}}

	s.muscleGroupRepo.EXPECT().List(mock.Anything).
		Return([]*domainexercise.MuscleGroup{{ID: 1, Name: "Chest"}, {ID: 2, Name: "Back"}}, nil)
	s.expectTx()
	s.exerciseRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(ex domainexercise.Exercise) bool {
			return ex.Name == "Bench Press" && ex.CreatedByUserID == userID
		})).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Bench Press"}, nil)
	s.exerciseRepo.EXPECT().BatchInsertMuscleGroups(mock.Anything, exerciseID, groups).Return(nil)

	created, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:           "Bench Press",
		Description:    "Chest press",
		Notes:          "Medium grip",
		UserID:         userID,
		MuscleGroupIDs: []int{1, 2},
	})
	s.Require().NoError(err)
	s.Equal(exerciseID, created.ID)
	s.Equal(groups, created.MuscleGroups)
	s.exerciseRepo.AssertCalled(s.T(), "BatchInsertMuscleGroups", mock.Anything, exerciseID, groups)
}

func (s *ServiceTestSuite) TestCreate_InvalidName() {
	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:   "   ",
		UserID: uuid.Must(uuid.NewV7()),
	})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidName)
	s.exerciseRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestCreate_InvalidUserID() {
	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{Name: "Squat"})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidUserID)
	s.exerciseRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestCreate_InvalidMuscleGroupID() {
	userID := uuid.Must(uuid.NewV7())
	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:           "Bench Press",
		UserID:         userID,
		MuscleGroupIDs: []int{0},
	})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidID)
	s.muscleGroupRepo.AssertNotCalled(s.T(), "List", mock.Anything)
}

func (s *ServiceTestSuite) TestCreate_MuscleGroupNotFound() {
	userID := uuid.Must(uuid.NewV7())
	s.muscleGroupRepo.EXPECT().List(mock.Anything).
		Return([]*domainexercise.MuscleGroup{{ID: 1, Name: "Chest"}}, nil)

	_, err := s.svc.Create(context.Background(), serviceexercise.CreateExerciseCommand{
		Name:           "Bench Press",
		UserID:         userID,
		MuscleGroupIDs: []int{1, 99},
	})
	s.Require().ErrorIs(err, domainexercise.ErrMuscleGroupMissing)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	newName := "Incline Bench Press"
	newGroups := []domainexercise.ExerciseMuscleGroup{{MuscleGroupID: 3}}

	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{
			ID:        exerciseID,
			Name:      "Bench Press",
			IsBuiltIn: false,
			Version:   3,
		}, nil)
	s.muscleGroupRepo.EXPECT().List(mock.Anything).
		Return([]*domainexercise.MuscleGroup{{ID: 3, Name: "Shoulders"}}, nil)
	s.expectTx()
	s.exerciseRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(ex domainexercise.Exercise) bool {
			return ex.ID == exerciseID && ex.Name == newName
		})).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: newName, Version: 4}, nil)
	s.exerciseRepo.EXPECT().ReplaceMuscleGroups(mock.Anything, exerciseID, newGroups).Return(nil)

	updated, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{
		ID:             exerciseID,
		Name:           &newName,
		MuscleGroupIDs: &[]int{3},
	})
	s.Require().NoError(err)
	s.Equal(newName, updated.Name)
	s.Equal(newGroups, updated.MuscleGroups)
}

func (s *ServiceTestSuite) TestUpdate_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(nil, domainexercise.ErrExerciseNotFound)

	_, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{ID: exerciseID})
	s.Require().ErrorIs(err, domainexercise.ErrExerciseNotFound)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_BuiltInRejected() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat", IsBuiltIn: true}, nil)

	_, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{
		ID:   exerciseID,
		Name: new("Front Squat"),
	})
	s.Require().ErrorIs(err, domainexercise.ErrCannotEditBuiltIn)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_InvalidName() {
	exerciseID := uuid.Must(uuid.NewV7())
	empty := "   "
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)

	_, err := s.svc.Update(context.Background(), serviceexercise.UpdateExerciseCommand{
		ID:   exerciseID,
		Name: &empty,
	})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidName)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestSoftDelete_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)
	s.exerciseRepo.EXPECT().SoftDelete(mock.Anything, exerciseID).Return(nil)

	err := s.svc.SoftDelete(context.Background(), exerciseID)
	s.NoError(err)
}

func (s *ServiceTestSuite) TestSoftDelete_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(nil, domainexercise.ErrExerciseNotFound)

	err := s.svc.SoftDelete(context.Background(), exerciseID)
	s.Require().ErrorIs(err, domainexercise.ErrExerciseNotFound)
	s.exerciseRepo.AssertNotCalled(s.T(), "SoftDelete", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestSoftDelete_BuiltInRejected() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat", IsBuiltIn: true}, nil)

	err := s.svc.SoftDelete(context.Background(), exerciseID)
	s.Require().ErrorIs(err, domainexercise.ErrCannotEditBuiltIn)
	s.exerciseRepo.AssertNotCalled(s.T(), "SoftDelete", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_Success() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{
			ID:   exerciseID,
			Name: "Squat",
			Media: []domainexercise.ExerciseMedia{
				{ID: uuid.Must(uuid.NewV7()), SortOrder: 0, S3Key: "exercises/first.jpg"},
				{ID: uuid.Must(uuid.NewV7()), SortOrder: 2, S3Key: "exercises/third.jpg"},
			},
		}, nil)
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return("exercises/generated-key")
	s.exerciseRepo.EXPECT().InsertMedia(mock.Anything, mock.MatchedBy(func(m domainexercise.ExerciseMedia) bool {
		return m.ID != uuid.Nil &&
			m.ExerciseID == exerciseID &&
			m.MediaType == domainexercise.MediaTypePhoto &&
			m.SortOrder == 3 &&
			m.S3Key == "exercises/generated-key"
	})).Return(nil)
	s.storage.EXPECT().
		PresignedPutURL(mock.Anything, "exercises/generated-key", "image/jpeg", uploadTTL).
		Return("https://presigned.example/upload", nil)

	got, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   domainexercise.MediaTypePhoto,
		ContentType: "image/jpeg",
	})
	s.Require().NoError(err)
	s.Equal(exerciseID, got.Media.ExerciseID)
	s.Equal(domainexercise.MediaTypePhoto, got.Media.MediaType)
	s.Equal("exercises/generated-key", got.Media.S3Key)
	s.Equal(3, got.Media.SortOrder)
	s.NotEqual(uuid.Nil, got.Media.ID)
	s.Equal("https://presigned.example/upload", got.UploadURL)
	s.Equal(uploadTTL, got.ExpiresIn)
}

func (s *ServiceTestSuite) TestUploadMedia_FirstItemGetsSortOrderZero() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return("exercises/generated-key")
	s.exerciseRepo.EXPECT().InsertMedia(mock.Anything, mock.MatchedBy(func(m domainexercise.ExerciseMedia) bool {
		return m.SortOrder == 0
	})).Return(nil)
	s.storage.EXPECT().
		PresignedPutURL(mock.Anything, "exercises/generated-key", "image/jpeg", uploadTTL).
		Return("https://presigned.example/upload", nil)

	got, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   domainexercise.MediaTypePhoto,
		ContentType: "image/jpeg",
	})
	s.Require().NoError(err)
	s.Equal(0, got.Media.SortOrder)
}

func (s *ServiceTestSuite) TestUploadMedia_NotFound() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(nil, domainexercise.ErrExerciseNotFound)

	_, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   domainexercise.MediaTypePhoto,
		ContentType: "image/jpeg",
	})
	s.Require().ErrorIs(err, domainexercise.ErrExerciseNotFound)
	s.exerciseRepo.AssertNotCalled(s.T(), "InsertMedia", mock.Anything, mock.Anything)
	s.storage.AssertNotCalled(s.T(), "GenerateKey", mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_BuiltInRejected() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat", IsBuiltIn: true}, nil)

	_, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   domainexercise.MediaTypePhoto,
		ContentType: "image/jpeg",
	})
	s.Require().ErrorIs(err, domainexercise.ErrCannotEditBuiltIn)
	s.exerciseRepo.AssertNotCalled(s.T(), "InsertMedia", mock.Anything, mock.Anything)
	s.storage.AssertNotCalled(s.T(), "GenerateKey", mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_InvalidMediaType() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)

	_, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   "gif",
		ContentType: "image/gif",
	})
	s.Require().ErrorIs(err, domainexercise.ErrInvalidMediaType)
	s.exerciseRepo.AssertNotCalled(s.T(), "InsertMedia", mock.Anything, mock.Anything)
	s.storage.AssertNotCalled(s.T(), "GenerateKey", mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_ContentTypeMismatch() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)

	_, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   domainexercise.MediaTypeVideo,
		ContentType: "image/jpeg",
	})
	s.Require().ErrorIs(err, domainexercise.ErrUnsupportedContentType)
	s.exerciseRepo.AssertNotCalled(s.T(), "InsertMedia", mock.Anything, mock.Anything)
	s.storage.AssertNotCalled(s.T(), "GenerateKey", mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_InsertFails() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return("exercises/generated-key")
	s.exerciseRepo.EXPECT().InsertMedia(mock.Anything, mock.Anything).
		Return(errors.New("insert failed"))

	_, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   domainexercise.MediaTypePhoto,
		ContentType: "image/jpeg",
	})
	s.Require().ErrorContains(err, "insert failed")
	s.storage.AssertNotCalled(s.T(), "PresignedPutURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_PresignFails() {
	exerciseID := uuid.Must(uuid.NewV7())
	s.exerciseRepo.EXPECT().FindByID(mock.Anything, exerciseID).
		Return(&domainexercise.Exercise{ID: exerciseID, Name: "Squat"}, nil)
	s.storage.EXPECT().GenerateKey("exercises/" + exerciseID.String()).Return("exercises/generated-key")
	s.exerciseRepo.EXPECT().InsertMedia(mock.Anything, mock.Anything).Return(nil)
	s.storage.EXPECT().
		PresignedPutURL(mock.Anything, "exercises/generated-key", "image/jpeg", uploadTTL).
		Return("", errors.New("presign failed"))

	_, err := s.svc.UploadMedia(context.Background(), serviceexercise.UploadExerciseMediaCommand{
		ExerciseID:  exerciseID,
		MediaType:   domainexercise.MediaTypePhoto,
		ContentType: "image/jpeg",
	})
	s.Require().ErrorContains(err, "presign failed")
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ServiceTestSuite))
}
