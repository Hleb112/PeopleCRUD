package service

import (
	"PeopleCRUD/internal/models"
	"PeopleCRUD/internal/repository"
	"PeopleCRUD/pkg/errors"
	"context"

	"github.com/sirupsen/logrus"
)

type PersonService interface {
	CreatePerson(ctx context.Context, req *models.CreatePersonRequest) (*models.PersonWithDetails, error)
	GetPersonByID(ctx context.Context, id int) (*models.PersonWithDetails, error)
	GetAllPeople(ctx context.Context, limit, offset int) ([]*models.PersonWithDetails, int, error)
	UpdatePerson(ctx context.Context, id int, req *models.UpdatePersonRequest) (*models.PersonWithDetails, error)
	DeletePerson(ctx context.Context, id int) error
	AddEmail(ctx context.Context, personID int, email string, isPrimary bool) error
	GetFriends(ctx context.Context, personID int) ([]models.Person, error)
	AddFriend(ctx context.Context, personID, friendID int) error
	RemoveFriend(ctx context.Context, personID, friendID int) error
}

type personService struct {
	repo   repository.PersonRepository
	logger *logrus.Logger
}

func NewPersonService(repo repository.PersonRepository, logger *logrus.Logger) PersonService {
	return &personService{
		repo:   repo,
		logger: logger,
	}
}

func (s *personService) CreatePerson(ctx context.Context, req *models.CreatePersonRequest) (*models.PersonWithDetails, error) {
	person := &models.Person{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		MiddleName: req.MiddleName,
	}

	if err := s.repo.Create(person); err != nil {
		return nil, errors.NewInternalServerError("Failed to create person")
	}

	if len(req.Emails) > 0 {
		for i, email := range req.Emails {
			isPrimary := i == 0
			if err := s.repo.AddEmail(person.ID, email, isPrimary); err != nil {
				s.logger.WithError(err).Error("Failed to add email")
			}
		}
	}

	return s.GetPersonByID(ctx, person.ID)
}

func (s *personService) GetPersonByID(ctx context.Context, id int) (*models.PersonWithDetails, error) {
	person, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	emails, _ := s.repo.GetEmails(id)
	friends, _ := s.repo.GetFriends(id)

	return &models.PersonWithDetails{
		Person:  *person,
		Emails:  emails,
		Friends: friends,
	}, nil
}

func (s *personService) GetAllPeople(ctx context.Context, limit, offset int) ([]*models.PersonWithDetails, int, error) {
	people, err := s.repo.GetAll(limit, offset)
	if err != nil {
		return nil, 0, errors.NewInternalServerError("Failed to get people")
	}

	total, err := s.repo.GetCount()
	if err != nil {
		return nil, 0, errors.NewInternalServerError("Failed to get total count")
	}

	result := make([]*models.PersonWithDetails, len(people))
	for i, person := range people {
		details, _ := s.GetPersonByID(ctx, person.ID)
		result[i] = details
	}

	return result, total, nil
}

func (s *personService) UpdatePerson(ctx context.Context, id int, req *models.UpdatePersonRequest) (*models.PersonWithDetails, error) {
	if _, err := s.repo.GetByID(id); err != nil {
		return nil, err
	}

	if err := s.repo.Update(id, req); err != nil {
		return nil, errors.NewInternalServerError("Failed to update person")
	}

	return s.GetPersonByID(ctx, id)
}

func (s *personService) DeletePerson(ctx context.Context, id int) error {
	if err := s.repo.Delete(id); err != nil {
		return errors.NewInternalServerError("Failed to delete person")
	}
	return nil
}

func (s *personService) AddEmail(ctx context.Context, personID int, email string, isPrimary bool) error {
	if _, err := s.repo.GetByID(personID); err != nil {
		return err
	}

	if isPrimary {
		emails, err := s.repo.GetEmails(personID)
		if err != nil {
			return errors.NewInternalServerError("Failed to get existing emails")
		}

		for _, e := range emails {
			if e.IsPrimary {
				if err := s.repo.UpdateEmail(e.ID, e.Email, false); err != nil {
					return errors.NewInternalServerError("Failed to update existing emails")
				}
			}
		}
	}

	if err := s.repo.AddEmail(personID, email, isPrimary); err != nil {
		return errors.NewInternalServerError("Failed to add email")
	}

	return nil
}

func (s *personService) GetFriends(ctx context.Context, personID int) ([]models.Person, error) {
	if _, err := s.repo.GetByID(personID); err != nil {
		return nil, err
	}

	return s.repo.GetFriends(personID)
}

func (s *personService) AddFriend(ctx context.Context, personID, friendID int) error {
	if personID == friendID {
		return errors.NewBadRequestError("Cannot add yourself as a friend")
	}

	// Проверяем что оба человека существуют
	if _, err := s.repo.GetByID(personID); err != nil {
		return err
	}

	if _, err := s.repo.GetByID(friendID); err != nil {
		return err
	}

	// Проверяем что дружба еще не существует
	friends, err := s.repo.GetFriends(personID)
	if err != nil {
		return err
	}

	for _, friend := range friends {
		if friend.ID == friendID {
			return errors.NewBadRequestError("Friendship already exists")
		}
	}

	// Добавляем дружбу в обе стороны
	if err := s.repo.AddFriend(personID, friendID); err != nil {
		return err
	}

	if err := s.repo.AddFriend(friendID, personID); err != nil {
		// Откатываем первую запись при ошибке
		s.repo.RemoveFriend(personID, friendID)
		return err
	}

	return nil
}

func (s *personService) RemoveFriend(ctx context.Context, personID, friendID int) error {
	// Удаляем дружбу в обе стороны
	if err := s.repo.RemoveFriend(personID, friendID); err != nil {
		return err
	}

	if err := s.repo.RemoveFriend(friendID, personID); err != nil {
		// Восстанавливаем первую запись при ошибке
		s.repo.AddFriend(personID, friendID)
		return err
	}

	return nil
}
