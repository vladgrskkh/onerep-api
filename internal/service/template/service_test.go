package template_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domaintemplate "github.com/vladgrskkh/onerep-api/internal/domain/template"
	servicetemplate "github.com/vladgrskkh/onerep-api/internal/service/template"
	templatemocks "github.com/vladgrskkh/onerep-api/internal/service/template/mocks"
)

type ServiceTestSuite struct {
	suite.Suite

	svc       *servicetemplate.TemplateService
	tmplRepo  *templatemocks.MockTemplateRepository
	trManager *templatemocks.MockTransactionManager
}

func (s *ServiceTestSuite) SetupTest() {
	s.tmplRepo = templatemocks.NewMockTemplateRepository(s.T())
	s.trManager = templatemocks.NewMockTransactionManager(s.T())
	s.svc = servicetemplate.NewTemplateService(s.tmplRepo, s.trManager)
}

// expectTx runs the transaction closure inline.
func (s *ServiceTestSuite) expectTx() {
	s.trManager.EXPECT().
		Do(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
		RunAndReturn(func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		})
}

func (s *ServiceTestSuite) TestList_Success() {
	userID := uuid.Must(uuid.NewV7())
	filter := domaintemplate.TemplateFilter{UserID: userID}
	expected := []*domaintemplate.Template{{ID: uuid.Must(uuid.NewV7()), Name: "Push Day"}}
	s.tmplRepo.EXPECT().List(mock.Anything, filter).Return(expected, nil)

	got, err := s.svc.List(context.Background(), filter)
	s.Require().NoError(err)
	s.Require().Len(got, 1)
	s.Equal(expected[0], got[0])
}

func (s *ServiceTestSuite) TestList_NilUserRejected() {
	_, err := s.svc.List(context.Background(), domaintemplate.TemplateFilter{})
	s.Require().ErrorIs(err, domaintemplate.ErrInvalidUserID)
	s.tmplRepo.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestList_UnscopedRejected() {
	_, err := s.svc.List(context.Background(), domaintemplate.TemplateFilter{})
	s.Require().ErrorIs(err, domaintemplate.ErrInvalidUserID)
	s.tmplRepo.AssertNotCalled(s.T(), "List", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestGet_PrivateOwned() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	expected := &domaintemplate.Template{ID: templateID, Name: "Push Day", CreatedByUserID: userID}
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).Return(expected, nil)

	got, err := s.svc.Get(context.Background(), templateID, userID)
	s.Require().NoError(err)
	s.Equal(expected, got)
}

func (s *ServiceTestSuite) TestGet_PublicByAnyone() {
	templateID := uuid.Must(uuid.NewV7())
	expected := &domaintemplate.Template{
		ID:              templateID,
		Name:            "Push Day",
		IsPublic:        true,
		CreatedByUserID: uuid.Must(uuid.NewV7()),
	}
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).Return(expected, nil)

	got, err := s.svc.Get(context.Background(), templateID, uuid.Must(uuid.NewV7()))
	s.Require().NoError(err)
	s.Equal(expected, got)
}

func (s *ServiceTestSuite) TestGet_PrivateByOther() {
	templateID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: uuid.Must(uuid.NewV7()),
		}, nil)

	_, err := s.svc.Get(context.Background(), templateID, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domaintemplate.ErrNotOwner)
}

func (s *ServiceTestSuite) TestGet_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	_, err := s.svc.Get(context.Background(), templateID, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domaintemplate.ErrTemplateNotFound)
}

func (s *ServiceTestSuite) TestCreate_Success() {
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())

	var templateID uuid.UUID
	s.expectTx()
	s.tmplRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(t domaintemplate.Template) bool {
			return t.Name == "Push Day" &&
				t.CreatedByUserID == userID &&
				!t.IsPublic &&
				t.Version == 1
		})).
		RunAndReturn(func(_ context.Context, t domaintemplate.Template) (*domaintemplate.Template, error) {
			templateID = t.ID
			return &t, nil
		})
	s.tmplRepo.EXPECT().
		BatchInsertExercises(mock.Anything, mock.MatchedBy(func(exercises []domaintemplate.TemplateExercise) bool {
			return len(exercises) == 1 &&
				exercises[0].TemplateID == templateID &&
				exercises[0].ExerciseID == exerciseID &&
				exercises[0].SortOrder == 1 &&
				exercises[0].PlannedSets == 3
		})).
		Return(nil)

	created, err := s.svc.Create(context.Background(), servicetemplate.CreateTemplateCommand{
		Name:        "Push Day",
		Description: "Chest, shoulders, triceps",
		UserID:      userID,
		Exercises: []servicetemplate.TemplateExerciseCommand{
			{ExerciseID: exerciseID, PlannedSets: 3},
		},
	})
	s.Require().NoError(err)
	s.Equal(templateID, created.ID)
	s.Equal("Push Day", created.Name)
	s.Require().Len(created.Exercises, 1)
	s.Equal(exerciseID, created.Exercises[0].ExerciseID)
}

func (s *ServiceTestSuite) TestCreate_NoExercises() {
	userID := uuid.Must(uuid.NewV7())
	templateID := uuid.Must(uuid.NewV7())

	s.expectTx()
	s.tmplRepo.EXPECT().Create(mock.Anything, mock.Anything).
		Return(&domaintemplate.Template{ID: templateID, Name: "Empty"}, nil)
	s.tmplRepo.EXPECT().BatchInsertExercises(mock.Anything, []domaintemplate.TemplateExercise{}).Return(nil)

	created, err := s.svc.Create(context.Background(), servicetemplate.CreateTemplateCommand{
		Name:   "Empty",
		UserID: userID,
	})
	s.Require().NoError(err)
	s.Equal(templateID, created.ID)
	s.Empty(created.Exercises)
}

func (s *ServiceTestSuite) TestCreate_InvalidName() {
	_, err := s.svc.Create(context.Background(), servicetemplate.CreateTemplateCommand{
		Name:   "   ",
		UserID: uuid.Must(uuid.NewV7()),
	})
	s.Require().ErrorIs(err, domaintemplate.ErrInvalidName)
	s.tmplRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestCreate_InvalidUserID() {
	_, err := s.svc.Create(context.Background(), servicetemplate.CreateTemplateCommand{Name: "Push Day"})
	s.Require().ErrorIs(err, domaintemplate.ErrInvalidUserID)
	s.tmplRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_Success() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())
	newName := "Heavy Push Day"

	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: userID,
			Version:         2,
		}, nil)
	s.expectTx()
	s.tmplRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(t domaintemplate.Template) bool {
			return t.ID == templateID && t.Name == newName
		})).
		Return(&domaintemplate.Template{ID: templateID, Name: newName, Version: 3}, nil)
	s.tmplRepo.EXPECT().
		ReplaceExercises(mock.Anything, templateID, mock.MatchedBy(func(exercises []domaintemplate.TemplateExercise) bool {
			return len(exercises) == 1 &&
				exercises[0].TemplateID == templateID &&
				exercises[0].ExerciseID == exerciseID &&
				exercises[0].SortOrder == 1
		})).
		Return(nil)

	updated, err := s.svc.Update(context.Background(), servicetemplate.UpdateTemplateCommand{
		ID:     templateID,
		UserID: userID,
		Name:   &newName,
		Exercises: &[]servicetemplate.TemplateExerciseCommand{
			{ExerciseID: exerciseID, PlannedSets: 5},
		},
	})
	s.Require().NoError(err)
	s.Equal(newName, updated.Name)
	s.Require().Len(updated.Exercises, 1)
	s.Equal(exerciseID, updated.Exercises[0].ExerciseID)
}

func (s *ServiceTestSuite) TestUpdate_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	otherUserID := uuid.Must(uuid.NewV7())
	newName := "Renamed"

	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: otherUserID,
		}, nil)

	_, err := s.svc.Update(context.Background(), servicetemplate.UpdateTemplateCommand{
		ID:     templateID,
		UserID: userID,
		Name:   &newName,
	})
	s.Require().ErrorIs(err, domaintemplate.ErrNotOwner)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
	s.tmplRepo.AssertNotCalled(s.T(), "Update", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	_, err := s.svc.Update(context.Background(), servicetemplate.UpdateTemplateCommand{
		ID:     templateID,
		UserID: uuid.Must(uuid.NewV7()),
	})
	s.Require().ErrorIs(err, domaintemplate.ErrTemplateNotFound)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUpdate_InvalidName() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	empty := "   "
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: userID,
		}, nil)

	_, err := s.svc.Update(context.Background(), servicetemplate.UpdateTemplateCommand{
		ID:     templateID,
		UserID: userID,
		Name:   &empty,
	})
	s.Require().ErrorIs(err, domaintemplate.ErrInvalidName)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestPublish_Success() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: userID,
			Version:         1,
		}, nil)
	s.tmplRepo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(t domaintemplate.Template) bool {
			return t.ID == templateID && t.IsPublic
		})).
		Return(&domaintemplate.Template{ID: templateID, Name: "Push Day", IsPublic: true, Version: 2}, nil)

	published, err := s.svc.Publish(context.Background(), templateID, userID)
	s.Require().NoError(err)
	s.True(published.IsPublic)
}

func (s *ServiceTestSuite) TestPublish_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: uuid.Must(uuid.NewV7()),
		}, nil)

	_, err := s.svc.Publish(context.Background(), templateID, userID)
	s.Require().ErrorIs(err, domaintemplate.ErrNotOwner)
	s.tmplRepo.AssertNotCalled(s.T(), "Update", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestFork_Success() {
	sourceID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())

	source := domaintemplate.Template{
		ID:              sourceID,
		Name:            "Push Day",
		Description:     "Chest, shoulders, triceps",
		IsPublic:        true,
		CreatedByUserID: uuid.Must(uuid.NewV7()),
		Exercises: []domaintemplate.TemplateExercise{
			{TemplateID: sourceID, ExerciseID: exerciseID, SortOrder: 1, PlannedSets: 3},
		},
		Media: []domaintemplate.TemplateMedia{
			{
				ID:         uuid.Must(uuid.NewV7()),
				TemplateID: sourceID,
				MediaType:  domaintemplate.MediaTypePhoto,
				SortOrder:  1,
				S3Key:      "templates/push.jpg",
			},
		},
	}
	s.tmplRepo.EXPECT().FindByID(mock.Anything, sourceID).Return(&source, nil)

	var forkID uuid.UUID
	s.expectTx()
	s.tmplRepo.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(t domaintemplate.Template) bool {
			forkID = t.ID
			return t.ID != sourceID &&
				t.Name == source.Name &&
				t.Description == source.Description &&
				t.CreatedByUserID == userID &&
				!t.IsPublic
		})).
		RunAndReturn(func(_ context.Context, t domaintemplate.Template) (*domaintemplate.Template, error) {
			return &t, nil
		})
	s.tmplRepo.EXPECT().
		BatchInsertExercises(mock.Anything, mock.MatchedBy(func(exercises []domaintemplate.TemplateExercise) bool {
			return len(exercises) == 1 &&
				exercises[0].TemplateID == forkID &&
				exercises[0].ExerciseID == exerciseID &&
				exercises[0].SortOrder == 1 &&
				exercises[0].PlannedSets == 3
		})).
		Return(nil)
	s.tmplRepo.EXPECT().
		BatchInsertMedia(mock.Anything, mock.MatchedBy(func(media []domaintemplate.TemplateMedia) bool {
			return len(media) == 1 &&
				media[0].TemplateID == forkID &&
				media[0].ID != source.Media[0].ID &&
				media[0].MediaType == domaintemplate.MediaTypePhoto &&
				media[0].SortOrder == 1 &&
				media[0].S3Key == "templates/push.jpg"
		})).
		Return(nil)

	fork, err := s.svc.Fork(context.Background(), sourceID, userID)
	s.Require().NoError(err)
	s.Equal(forkID, fork.ID)
	s.Equal(userID, fork.CreatedByUserID)
	s.Require().Len(fork.Exercises, 1)
	s.Require().Len(fork.Media, 1)
}

func (s *ServiceTestSuite) TestFork_PrivateByOwner() {
	sourceID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	exerciseID := uuid.Must(uuid.NewV7())

	source := domaintemplate.Template{
		ID:              sourceID,
		Name:            "Push Day",
		CreatedByUserID: userID,
		Exercises: []domaintemplate.TemplateExercise{
			{TemplateID: sourceID, ExerciseID: exerciseID, SortOrder: 1, PlannedSets: 3},
		},
	}
	s.tmplRepo.EXPECT().FindByID(mock.Anything, sourceID).Return(&source, nil)

	s.expectTx()
	s.tmplRepo.EXPECT().Create(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, t domaintemplate.Template) (*domaintemplate.Template, error) {
			return &t, nil
		})
	s.tmplRepo.EXPECT().
		BatchInsertExercises(mock.Anything, mock.AnythingOfType("[]template.TemplateExercise")).
		Return(nil)
	s.tmplRepo.EXPECT().BatchInsertMedia(mock.Anything, mock.AnythingOfType("[]template.TemplateMedia")).Return(nil)

	fork, err := s.svc.Fork(context.Background(), sourceID, userID)
	s.Require().NoError(err)
	s.Equal(userID, fork.CreatedByUserID)
	s.Require().Len(fork.Exercises, 1)
}

func (s *ServiceTestSuite) TestFork_PrivateByOther() {
	sourceID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, sourceID).
		Return(&domaintemplate.Template{
			ID:              sourceID,
			Name:            "Push Day",
			CreatedByUserID: uuid.Must(uuid.NewV7()),
		}, nil)

	_, err := s.svc.Fork(context.Background(), sourceID, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domaintemplate.ErrNotOwner)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
	s.tmplRepo.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestFork_NotFound() {
	sourceID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, sourceID).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	_, err := s.svc.Fork(context.Background(), sourceID, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domaintemplate.ErrTemplateNotFound)
	s.trManager.AssertNotCalled(s.T(), "Do", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestSoftDelete_Success() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: userID,
		}, nil)
	s.tmplRepo.EXPECT().SoftDelete(mock.Anything, templateID).Return(nil)

	err := s.svc.SoftDelete(context.Background(), templateID, userID)
	s.NoError(err)
}

func (s *ServiceTestSuite) TestUploadMedia_Success() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: userID,
			Media: []domaintemplate.TemplateMedia{
				{ID: uuid.Must(uuid.NewV7()), SortOrder: 1, S3Key: "templates/cover.jpg"},
			},
		}, nil)
	s.tmplRepo.EXPECT().InsertMedia(mock.Anything, mock.MatchedBy(func(m domaintemplate.TemplateMedia) bool {
		return m.ID != uuid.Nil &&
			m.TemplateID == templateID &&
			m.MediaType == domaintemplate.MediaTypePhoto &&
			m.SortOrder == 2 &&
			m.S3Key == "templates/push-day.jpg"
	})).Return(nil)

	got, err := s.svc.UploadMedia(context.Background(), servicetemplate.UploadTemplateMediaCommand{
		TemplateID: templateID,
		UserID:     userID,
		S3Key:      "templates/push-day.jpg",
	})
	s.Require().NoError(err)
	s.Equal(templateID, got.TemplateID)
	s.Equal(domaintemplate.MediaTypePhoto, got.MediaType)
	s.Equal("templates/push-day.jpg", got.S3Key)
	s.Equal(2, got.SortOrder)
	s.NotEqual(uuid.Nil, got.ID)
}

func (s *ServiceTestSuite) TestUploadMedia_NotFound() {
	templateID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(nil, domaintemplate.ErrTemplateNotFound)

	_, err := s.svc.UploadMedia(context.Background(), servicetemplate.UploadTemplateMediaCommand{
		TemplateID: templateID,
		UserID:     uuid.Must(uuid.NewV7()),
		S3Key:      "templates/push-day.jpg",
	})
	s.Require().ErrorIs(err, domaintemplate.ErrTemplateNotFound)
	s.tmplRepo.AssertNotCalled(s.T(), "InsertMedia", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: uuid.Must(uuid.NewV7()),
		}, nil)

	_, err := s.svc.UploadMedia(context.Background(), servicetemplate.UploadTemplateMediaCommand{
		TemplateID: templateID,
		UserID:     uuid.Must(uuid.NewV7()),
		S3Key:      "templates/push-day.jpg",
	})
	s.Require().ErrorIs(err, domaintemplate.ErrNotOwner)
	s.tmplRepo.AssertNotCalled(s.T(), "InsertMedia", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_EmptyS3Key() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: userID,
		}, nil)

	_, err := s.svc.UploadMedia(context.Background(), servicetemplate.UploadTemplateMediaCommand{
		TemplateID: templateID,
		UserID:     userID,
	})
	s.Require().ErrorIs(err, domaintemplate.ErrInvalidS3Key)
	s.tmplRepo.AssertNotCalled(s.T(), "InsertMedia", mock.Anything, mock.Anything)
}

func (s *ServiceTestSuite) TestUploadMedia_InsertFails() {
	templateID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: userID,
		}, nil)
	s.tmplRepo.EXPECT().InsertMedia(mock.Anything, mock.Anything).
		Return(errors.New("insert failed"))

	_, err := s.svc.UploadMedia(context.Background(), servicetemplate.UploadTemplateMediaCommand{
		TemplateID: templateID,
		UserID:     userID,
		S3Key:      "templates/push-day.jpg",
	})
	s.Require().ErrorContains(err, "insert failed")
}

func (s *ServiceTestSuite) TestSoftDelete_NotOwner() {
	templateID := uuid.Must(uuid.NewV7())
	s.tmplRepo.EXPECT().FindByID(mock.Anything, templateID).
		Return(&domaintemplate.Template{
			ID:              templateID,
			Name:            "Push Day",
			CreatedByUserID: uuid.Must(uuid.NewV7()),
		}, nil)

	err := s.svc.SoftDelete(context.Background(), templateID, uuid.Must(uuid.NewV7()))
	s.Require().ErrorIs(err, domaintemplate.ErrNotOwner)
	s.tmplRepo.AssertNotCalled(s.T(), "SoftDelete", mock.Anything, mock.Anything)
}

func TestServiceSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ServiceTestSuite))
}
