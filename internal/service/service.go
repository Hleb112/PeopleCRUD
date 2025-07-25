package service

import (
	"PeopleCRUD/internal/cache"
	"PeopleCRUD/internal/models"
	"PeopleCRUD/internal/repository"
	"PeopleCRUD/pkg/errors"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type PersonService interface {
	CreatePerson(ctx context.Context, req *models.CreatePersonRequest) (*models.PersonWithDetails, error)
	GetPersonByID(ctx context.Context, id int) (*models.PersonWithDetails, error)
	GetPeopleByLastName(ctx context.Context, lastName string) ([]*models.PersonWithDetails, error)
	GetAllPeople(ctx context.Context, limit, offset int) ([]*models.PersonWithDetails, int, error)
	UpdatePerson(ctx context.Context, id int, req *models.UpdatePersonRequest) (*models.PersonWithDetails, error)
	DeletePerson(ctx context.Context, id int) error
	AddEmail(ctx context.Context, personID int, email string, isPrimary bool) error
	AddFriend(ctx context.Context, personID, friendID int) error
	GetFriends(ctx context.Context, personID int) ([]models.Person, error)
	RemoveFriend(ctx context.Context, personID, friendID int) error
}

type personService struct {
	repo   repository.PersonRepository
	cache  *cache.MemoryCache
	logger *logrus.Logger
}

func NewPersonService(repo repository.PersonRepository, cache *cache.MemoryCache, logger *logrus.Logger) PersonService {
	return &personService{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (s *personService) CreatePerson(ctx context.Context, req *models.CreatePersonRequest) (*models.PersonWithDetails, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	person := &models.Person{
		FirstName:  strings.TrimSpace(req.FirstName),
		LastName:   strings.TrimSpace(req.LastName),
		MiddleName: req.MiddleName,
	}

	// Создаем человека с транзакцией для email
	if len(req.Emails) > 0 {
		err := s.repo.CreateWithTransaction(person, req.Emails)
		if err != nil {
			s.logger.WithError(err).Error("Failed to create person with emails")
			return nil, errors.NewInternalServerError(err.Error())
		}
	} else {
		if err := s.repo.Create(person); err != nil {
			s.logger.WithError(err).Error("Failed to create person")
			return nil, errors.NewInternalServerError(err.Error())
		}
	}

	return s.GetPersonByID(ctx, person.ID)
}

func (s *personService) GetPersonByID(ctx context.Context, id int) (*models.PersonWithDetails, error) {
	cacheKey := fmt.Sprintf("person:%d", id)

	if cached, found := s.cache.Get(cacheKey); found {
		if person, ok := cached.(*models.PersonWithDetails); ok {
			s.logger.Debugf("Returning person %d from cache", id)
			return person, nil
		}
	}

	person, err := s.repo.GetByID(id)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get person by ID")
		return nil, errors.NewInternalServerError(err.Error())
	}

	emails, err := s.repo.GetEmails(person.ID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get person emails")
		emails = []models.Email{}
	}

	friends, err := s.repo.GetFriends(person.ID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get person friends")
		friends = []models.Person{}
	}

	result := &models.PersonWithDetails{
		Person:  *person,
		Emails:  emails,
		Friends: friends,
	}

	s.cache.Set(cacheKey, result, 5*time.Minute)
	return result, nil
}

func (s *personService) GetPeopleByLastName(ctx context.Context, lastName string) ([]*models.PersonWithDetails, error) {
	if strings.TrimSpace(lastName) == "" {
		return nil, errors.NewValidationError("Last name cannot be empty")
	}

	people, err := s.repo.GetByLastName(lastName)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get people by last name")
		return nil, errors.NewInternalServerError(err.Error())
	}

	result := make([]*models.PersonWithDetails, len(people))
	for i, person := range people {
		details, err := s.GetPersonByID(ctx, person.ID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get person details")
			continue
		}
		result[i] = details
	}

	return result, nil
}

func (s *personService) GetAllPeople(ctx context.Context, limit, offset int) ([]*models.PersonWithDetails, int, error) {
	cacheKey := fmt.Sprintf("people:limit=%d&offset=%d", limit, offset)

	if cached, found := s.cache.Get(cacheKey); found {
		if data, ok := cached.(struct {
			People []*models.PersonWithDetails
			Total  int
		}); ok {
			s.logger.Debug("Returning people list from cache")
			return data.People, data.Total, nil
		}
	}

	people, err := s.repo.GetAll(limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get all people")
		return nil, 0, errors.NewInternalServerError(err.Error())
	}

	total, err := s.repo.GetCount()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get people count")
		return nil, 0, errors.NewInternalServerError("Failed to get people count")
	}

	result := make([]*models.PersonWithDetails, len(people))
	for i, person := range people {
		details, err := s.GetPersonByID(ctx, person.ID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get person details")
			continue
		}
		result[i] = details
	}

	cacheData := struct {
		People []*models.PersonWithDetails
		Total  int
	}{
		People: result,
		Total:  total,
	}

	s.cache.Set(cacheKey, cacheData, 1*time.Minute)
	return result, total, nil
}

func (s *personService) UpdatePerson(ctx context.Context, id int, req *models.UpdatePersonRequest) (*models.PersonWithDetails, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if _, err := s.repo.GetByID(id); err != nil {
		s.logger.WithError(err).Error("Failed to check person existence")
		return nil, errors.NewInternalServerError(err.Error())
	}

	if err := s.repo.Update(id, req); err != nil {
		s.logger.WithError(err).Error("Failed to update person")
		return nil, errors.NewInternalServerError("Failed to update person")
	}

	s.invalidatePersonCache(id)
	return s.GetPersonByID(ctx, id)
}

func (s *personService) DeletePerson(ctx context.Context, id int) error {
	if _, err := s.repo.GetByID(id); err != nil {
		s.logger.WithError(err).Error("Failed to check person existence")
		return errors.NewInternalServerError("Failed to delete person")
	}

	if err := s.repo.Delete(id); err != nil {
		s.logger.WithError(err).Error("Failed to delete person")
		return errors.NewInternalServerError(err.Error())
	}

	s.invalidatePersonCache(id)
	return nil
}

func (s *personService) AddEmail(ctx context.Context, personID int, email string, isPrimary bool) error {
	if _, err := s.repo.GetByID(personID); err != nil {
		s.logger.WithError(err).Error("Failed to check person existence")
		return errors.NewInternalServerError("Failed to add email")
	}

	if isPrimary {
		emails, err := s.repo.GetEmails(personID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get person emails")
			return errors.NewInternalServerError("Failed to update emails")
		}

		for _, e := range emails {
			if e.IsPrimary {
				if err := s.repo.UpdateEmail(e.ID, e.Email, false); err != nil {
					s.logger.WithError(err).Error("Failed to update email")
					return errors.NewInternalServerError("Failed to update emails")
				}
			}
		}
	}

	if err := s.repo.AddEmail(personID, email, isPrimary); err != nil {
		s.logger.WithError(err).Error("Failed to add email")
		return errors.NewInternalServerError("Failed to add email")
	}

	return nil
}

func (s *personService) AddFriend(ctx context.Context, personID, friendID int) error {
	if personID == friendID {
		return errors.NewValidationError("Cannot add yourself as a friend")
	}

	if _, err := s.repo.GetByID(personID); err != nil {
		s.logger.WithError(err).Error("Failed to check person existence")
		return errors.NewInternalServerError("Failed to add friend")
	}

	if _, err := s.repo.GetByID(friendID); err != nil {
		s.logger.WithError(err).Error("Failed to check friend existence")
		return errors.NewInternalServerError("Failed to add friend")
	}

	friends, err := s.repo.GetFriends(personID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get friends list")
		return errors.NewInternalServerError("Failed to add friend")
	}

	for _, friend := range friends {
		if friend.ID == friendID {
			return errors.NewValidationError("Friendship already exists")
		}
	}

	if err := s.repo.AddFriend(personID, friendID); err != nil {
		s.logger.WithError(err).Error("Failed to add friend")
		return errors.NewInternalServerError("Failed to add friend")
	}

	if err := s.repo.AddFriend(friendID, personID); err != nil {
		s.repo.RemoveFriend(personID, friendID)
		s.logger.WithError(err).Error("Failed to add reciprocal friendship")
		return errors.NewInternalServerError("Failed to add friend")
	}

	return nil
}

func (s *personService) GetFriends(ctx context.Context, personID int) ([]models.Person, error) {
	if _, err := s.repo.GetByID(personID); err != nil {
		s.logger.WithError(err).Error("Failed to check person existence")
		return nil, errors.NewInternalServerError("Failed to get friends")
	}

	friends, err := s.repo.GetFriends(personID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get friends")
		return nil, errors.NewInternalServerError("Failed to get friends")
	}

	return friends, nil
}

func (s *personService) RemoveFriend(ctx context.Context, personID, friendID int) error {
	if err := s.repo.RemoveFriend(personID, friendID); err != nil {
		s.logger.WithError(err).Error("Failed to remove friend")
		return errors.NewInternalServerError("Failed to remove friend")
	}

	if err := s.repo.RemoveFriend(friendID, personID); err != nil {
		s.repo.AddFriend(personID, friendID)
		s.logger.WithError(err).Error("Failed to remove reciprocal friendship")
		return errors.NewInternalServerError("Failed to remove friend")
	}

	return nil
}

func (s *personService) invalidatePersonCache(id int) {
	s.cache.Delete(fmt.Sprintf("person:%d", id))
	s.cache.DeleteByPrefix("people:")
}
