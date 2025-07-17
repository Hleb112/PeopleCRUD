package service

import (
	"PeopleCRUD/internal/cache"
	"PeopleCRUD/internal/models"
	"PeopleCRUD/internal/repository"
	"PeopleCRUD/internal/service/external"
	"PeopleCRUD/pkg/errors"
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type PersonService interface {
	CreatePerson(ctx context.Context, req *models.CreatePersonRequest) (*models.PersonWithDetails, error)
	GetPersonByID(ctx context.Context, id int) (*models.PersonWithDetails, error)
	GetPeopleByLastName(ctx context.Context, lastName string) ([]*models.PersonWithDetails, error)
	GetAllPeople(ctx context.Context, page, limit int) ([]*models.PersonWithDetails, int, error)
	UpdatePerson(ctx context.Context, id int, req *models.UpdatePersonRequest) (*models.PersonWithDetails, error)
	DeletePerson(ctx context.Context, id int) error
	AddEmail(ctx context.Context, personID int, email string, isPrimary bool) error
	UpdateEmail(ctx context.Context, emailID int, email string, isPrimary bool) error
	DeleteEmail(ctx context.Context, emailID int) error
	AddFriend(ctx context.Context, personID, friendID int) error
	GetFriends(ctx context.Context, personID int) ([]*models.Person, error)
	RemoveFriend(ctx context.Context, personID, friendID int) error
}

type personService struct {
	repo              repository.PersonRepository
	cache             *cache.MemoryCache
	logger            *logrus.Logger
	agifyClient       *external.AgifyClient
	genderizeClient   *external.GenderizeClient
	nationalizeClient *external.NationalizeClient
}

func NewPersonService(repo repository.PersonRepository, cache *cache.MemoryCache, logger *logrus.Logger) PersonService {
	return &personService{
		repo:              repo,
		cache:             cache,
		logger:            logger,
		agifyClient:       external.NewAgifyClient(),
		genderizeClient:   external.NewGenderizeClient(),
		nationalizeClient: external.NewNationalizeClient(),
	}
}

func (s *personService) CreatePerson(ctx context.Context, req *models.CreatePersonRequest) (*models.PersonWithDetails, error) {
	s.logger.WithFields(logrus.Fields{
		"first_name": req.FirstName,
		"last_name":  req.LastName,
	}).Info("Creating new person")

	// Создаем базовую структуру человека
	person := &models.Person{
		FirstName:  req.FirstName,
		LastName:   req.LastName,
		MiddleName: req.MiddleName,
	}

	// Получаем данные от внешних API параллельно
	if err := s.enrichPersonData(ctx, person); err != nil {
		s.logger.WithError(err).Warn("Failed to enrich person data from external APIs")
	}

	// Создаем человека в базе данных
	if err := s.repo.Create(person); err != nil {
		s.logger.WithError(err).Error("Failed to create person in database")
		return nil, errors.NewInternalServerError("Failed to create person")
	}

	// Добавляем email-адреса, если они указаны
	if len(req.Emails) > 0 {
		for i, email := range req.Emails {
			isPrimary := i == 0 // Первый email делаем основным
			if err := s.repo.AddEmail(person.ID, email, isPrimary); err != nil {
				s.logger.WithError(err).Error("Failed to add email")
				// Не возвращаем ошибку, так как человек уже создан
			}
		}
	}

	// Очищаем кэш
	s.clearPersonCache(person.ID)

	return s.GetPersonByID(ctx, person.ID)
}

func (s *personService) enrichPersonData(ctx context.Context, person *models.Person) error {
	g, ctx := errgroup.WithContext(ctx)

	// Получаем возраст
	g.Go(func() error {
		age, err := s.agifyClient.GetAge(ctx, person.FirstName)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to get age from Agify")
			return nil // Не прерываем выполнение
		}
		person.Age = age
		return nil
	})

	// Получаем пол
	g.Go(func() error {
		gender, err := s.genderizeClient.GetGender(ctx, person.FirstName)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to get gender from Genderize")
			return nil
		}
		person.Gender = gender
		return nil
	})

	// Получаем национальность
	g.Go(func() error {
		nationality, err := s.nationalizeClient.GetNationality(ctx, person.FirstName)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to get nationality from Nationalize")
			return nil
		}
		person.Nationality = nationality
		return nil
	})

	return g.Wait()
}

func (s *personService) GetPersonByID(ctx context.Context, id int) (*models.PersonWithDetails, error) {
	cacheKey := fmt.Sprintf("person_%d", id)

	// Проверяем кэш
	if cached, found := s.cache.Get(cacheKey); found {
		if person, ok := cached.(*models.PersonWithDetails); ok {
			s.logger.WithField("person_id", id).Debug("Person found in cache")
			return person, nil
		}
	}

	// Получаем из базы данных
	person, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Получаем дополнительные данные параллельно
	result := &models.PersonWithDetails{Person: *person}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		emails, err := s.repo.GetEmails(id)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get emails")
			return nil
		}
		result.Emails = emails
		return nil
	})

	g.Go(func() error {
		friends, err := s.repo.GetFriends(id)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get friends")
			return nil
		}
		result.Friends = friends
		return nil
	})

	if err := g.Wait(); err != nil {
		s.logger.WithError(err).Error("Failed to get person details")
	}

	// Кэшируем на 5 минут
	s.cache.Set(cacheKey, result, 5*time.Minute)

	return result, nil
}

func (s *personService) GetPeopleByLastName(ctx context.Context, lastName string) ([]*models.PersonWithDetails, error) {
	cacheKey := fmt.Sprintf("people_by_lastname_%s", lastName)

	// Проверяем кэш
	if cached, found := s.cache.Get(cacheKey); found {
		if people, ok := cached.([]*models.PersonWithDetails); ok {
			s.logger.WithField("last_name", lastName).Debug("People found in cache")
			return people, nil
		}
	}

	people, err := s.repo.GetByLastName(lastName)
	if err != nil {
		return nil, err
	}

	// Получаем детали для каждого человека параллельно
	result := make([]*models.PersonWithDetails, len(people))
	g, ctx := errgroup.WithContext(ctx)

	for i, person := range people {
		i, person := i, person
		g.Go(func() error {
			details, err := s.GetPersonByID(ctx, person.ID)
			if err != nil {
				s.logger.WithError(err).Error("Failed to get person details")
				result[i] = &models.PersonWithDetails{Person: *person}
				return nil
			}
			result[i] = details
			return nil
		})
	}

	g.Wait()

	// Кэшируем на 2 минуты
	s.cache.Set(cacheKey, result, 2*time.Minute)

	return result, nil
}

func (s *personService) GetAllPeople(ctx context.Context, page, limit int) ([]*models.PersonWithDetails, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	cacheKey := fmt.Sprintf("people_page_%d_limit_%d", page, limit)

	// Проверяем кэш
	if cached, found := s.cache.Get(cacheKey); found {
		if result, ok := cached.(map[string]interface{}); ok {
			people := result["people"].([]*models.PersonWithDetails)
			total := result["total"].(int)
			s.logger.WithFields(logrus.Fields{
				"page":  page,
				"limit": limit,
			}).Debug("People page found in cache")
			return people, total, nil
		}
	}

	// Получаем общее количество
	total, err := s.repo.GetCount()
	if err != nil {
		return nil, 0, errors.NewInternalServerError("Failed to get total count")
	}

	// Получаем людей
	people, err := s.repo.GetAll(limit, offset)
	if err != nil {
		return nil, 0, errors.NewInternalServerError("Failed to get people")
	}

	// Получаем детали для каждого человека параллельно
	result := make([]*models.PersonWithDetails, len(people))
	g, ctx := errgroup.WithContext(ctx)

	for i, person := range people {
		i, person := i, person
		g.Go(func() error {
			details, err := s.GetPersonByID(ctx, person.ID)
			if err != nil {
				s.logger.WithError(err).Error("Failed to get person details")
				result[i] = &models.PersonWithDetails{Person: *person}
				return nil
			}
			result[i] = details
			return nil
		})
	}

	g.Wait()

	// Кэшируем на 1 минуту
	cacheData := map[string]interface{}{
		"people": result,
		"total":  total,
	}
	s.cache.Set(cacheKey, cacheData, 1*time.Minute)

	return result, total, nil
}

func (s *personService) UpdatePerson(ctx context.Context, id int, req *models.UpdatePersonRequest) (*models.PersonWithDetails, error) {
	// Проверяем, что человек существует
	_, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Обновляем
	if err := s.repo.Update(id, req); err != nil {
		return nil, err
	}

	// Очищаем кэш
	s.clearPersonCache(id)

	return s.GetPersonByID(ctx, id)
}

func (s *personService) DeletePerson(ctx context.Context, id int) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// Очищаем кэш
	s.clearPersonCache(id)

	return nil
}

func (s *personService) AddEmail(ctx context.Context, personID int, email string, isPrimary bool) error {
	// Проверяем, что человек существует
	_, err := s.repo.GetByID(personID)
	if err != nil {
		return err
	}

	if err := s.repo.AddEmail(personID, email, isPrimary); err != nil {
		return errors.NewInternalServerError("Failed to add email")
	}

	// Очищаем кэш
	s.clearPersonCache(personID)

	return nil
}

func (s *personService) UpdateEmail(ctx context.Context, emailID int, email string, isPrimary bool) error {
	if err := s.repo.UpdateEmail(emailID, email, isPrimary); err != nil {
		return err
	}

	// Очищаем весь кэш (так как не знаем конкретный person_id)
	s.clearAllCache()

	return nil
}

func (s *personService) DeleteEmail(ctx context.Context, emailID int) error {
	if err := s.repo.DeleteEmail(emailID); err != nil {
		return err
	}

	// Очищаем весь кэш
	s.clearAllCache()

	return nil
}

func (s *personService) AddFriend(ctx context.Context, personID, friendID int) error {
	if personID == friendID {
		return errors.NewBadRequestError("Cannot add yourself as a friend")
	}

	// Проверяем, что оба человека существуют
	_, err := s.repo.GetByID(personID)
	if err != nil {
		return err
	}

	_, err = s.repo.GetByID(friendID)
	if err != nil {
		return err
	}

	if err := s.repo.AddFriend(personID, friendID); err != nil {
		return errors.NewInternalServerError("Failed to add friend")
	}

	// Очищаем кэш для обоих людей
	s.clearPersonCache(personID)
	s.clearPersonCache(friendID)

	return nil
}

func (s *personService) GetFriends(ctx context.Context, personID int) ([]*models.Person, error) {
	// Проверяем, что человек существует
	_, err := s.repo.GetByID(personID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetFriends(personID)
}

func (s *personService) RemoveFriend(ctx context.Context, personID, friendID int) error {
	if err := s.repo.RemoveFriend(personID, friendID); err != nil {
		return errors.NewInternalServerError("Failed to remove friend")
	}

	// Очищаем кэш для обоих людей
	s.clearPersonCache(personID)
	s.clearPersonCache(friendID)

	return nil
}

func (s *personService) clearPersonCache(personID int) {
	cacheKey := fmt.Sprintf("person_%d", personID)
	s.cache.Delete(cacheKey)
}

func (s *personService) clearAllCache() {
	// Простая реализация - создаем новый кэш
	// В продакшене лучше использовать более сложную логику
	*s.cache = *cache.NewMemoryCache()
}
