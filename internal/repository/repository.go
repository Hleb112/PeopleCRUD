package repository

import (
	"PeopleCRUD/internal/models"
	"PeopleCRUD/pkg/errors"
	"database/sql"
	"fmt"
	"strings"
)

type PersonRepository interface {
	Create(person *models.Person) error
	CreateWithTransaction(person *models.Person, emails []string) error
	GetByID(id int) (*models.Person, error)
	GetByLastName(lastName string) ([]*models.Person, error)
	GetAll(limit, offset int) ([]*models.Person, error)
	GetCount() (int, error)
	Update(id int, req *models.UpdatePersonRequest) error
	Delete(id int) error
	AddEmail(personID int, email string, isPrimary bool) error
	UpdateEmail(emailID int, email string, isPrimary bool) error
	DeleteEmail(emailID int) error
	GetEmails(personID int) ([]models.Email, error)
	AddFriend(personID, friendID int) error
	RemoveFriend(personID, friendID int) error
	GetFriends(personID int) ([]models.Person, error)
}

type personRepository struct {
	db *sql.DB
}

func NewPersonRepository(db *sql.DB) PersonRepository {
	return &personRepository{db: db}
}

func (r *personRepository) CreateWithTransaction(person *models.Person, emails []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Создание человека
	query := `
		INSERT INTO people (first_name, last_name, middle_name, age, gender, nationality)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err = tx.QueryRow(query, person.FirstName, person.LastName, person.MiddleName,
		person.Age, person.Gender, person.Nationality).Scan(&person.ID, &person.CreatedAt, &person.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create person: %w", err)
	}

	// Добавление email'ов
	for i, email := range emails {
		isPrimary := i == 0
		emailQuery := `INSERT INTO emails (person_id, email, is_primary) VALUES ($1, $2, $3)`
		if _, err := tx.Exec(emailQuery, person.ID, email, isPrimary); err != nil {
			return fmt.Errorf("failed to add email: %w", err)
		}
	}

	return tx.Commit()
}

func (r *personRepository) Create(person *models.Person) error {
	query := `
		INSERT INTO people (first_name, last_name, middle_name, age, gender, nationality)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(query, person.FirstName, person.LastName, person.MiddleName,
		person.Age, person.Gender, person.Nationality).Scan(&person.ID, &person.CreatedAt, &person.UpdatedAt)
	if err != nil {
		return errors.NewInternalServerError("Failed to create person")
	}
	return nil
}

func (r *personRepository) GetByID(id int) (*models.Person, error) {
	query := `
		SELECT id, first_name, last_name, middle_name, age, gender, nationality, created_at, updated_at
		FROM people WHERE id = $1`

	person := &models.Person{}
	err := r.db.QueryRow(query, id).Scan(
		&person.ID, &person.FirstName, &person.LastName, &person.MiddleName,
		&person.Age, &person.Gender, &person.Nationality, &person.CreatedAt, &person.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFoundError("Person not found")
		}
		return nil, errors.NewInternalServerError("Failed to get person")
	}
	return person, nil
}

func (r *personRepository) Update(id int, req *models.UpdatePersonRequest) error {
	// Проверяем, что человек существует
	_, err := r.GetByID(id)
	if err != nil {
		return err
	}

	// Строим динамический запрос
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.FirstName != nil {
		setParts = append(setParts, fmt.Sprintf("first_name = $%d", argIndex))
		args = append(args, *req.FirstName)
		argIndex++
	}
	if req.LastName != nil {
		setParts = append(setParts, fmt.Sprintf("last_name = $%d", argIndex))
		args = append(args, *req.LastName)
		argIndex++
	}
	if req.MiddleName != nil {
		setParts = append(setParts, fmt.Sprintf("middle_name = $%d", argIndex))
		args = append(args, *req.MiddleName)
		argIndex++
	}
	if req.Age != nil {
		setParts = append(setParts, fmt.Sprintf("age = $%d", argIndex))
		args = append(args, *req.Age)
		argIndex++
	}
	if req.Gender != nil {
		setParts = append(setParts, fmt.Sprintf("gender = $%d", argIndex))
		args = append(args, *req.Gender)
		argIndex++
	}
	if req.Nationality != nil {
		setParts = append(setParts, fmt.Sprintf("nationality = $%d", argIndex))
		args = append(args, *req.Nationality)
		argIndex++
	}

	if len(setParts) == 0 {
		return errors.NewValidationError("No fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE people 
		SET %s, updated_at = CURRENT_TIMESTAMP
		WHERE id = $%d`,
		strings.Join(setParts, ", "), argIndex)

	args = append(args, id)

	_, err = r.db.Exec(query, args...)
	if err != nil {
		return errors.NewInternalServerError("Failed to update person")
	}

	return nil
}

func (r *personRepository) GetByLastName(lastName string) ([]*models.Person, error) {
	query := `
		SELECT id, first_name, last_name, middle_name, age, gender, nationality, created_at, updated_at
		FROM people WHERE last_name = $1`

	rows, err := r.db.Query(query, lastName)
	if err != nil {
		return nil, errors.NewInternalServerError("Failed to get people by last name")
	}
	defer rows.Close()

	var people []*models.Person
	for rows.Next() {
		person := &models.Person{}
		err := rows.Scan(
			&person.ID, &person.FirstName, &person.LastName, &person.MiddleName,
			&person.Age, &person.Gender, &person.Nationality, &person.CreatedAt, &person.UpdatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalServerError("Failed to scan person")
		}
		people = append(people, person)
	}

	return people, nil
}

func (r *personRepository) GetAll(limit, offset int) ([]*models.Person, error) {
	query := `
		SELECT id, first_name, last_name, middle_name, age, gender, nationality, created_at, updated_at
		FROM people ORDER BY id LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, errors.NewInternalServerError("Failed to get all people")
	}
	defer rows.Close()

	var people []*models.Person
	for rows.Next() {
		person := &models.Person{}
		err := rows.Scan(
			&person.ID, &person.FirstName, &person.LastName, &person.MiddleName,
			&person.Age, &person.Gender, &person.Nationality, &person.CreatedAt, &person.UpdatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalServerError("Failed to scan person")
		}
		people = append(people, person)
	}

	return people, nil
}

func (r *personRepository) GetCount() (int, error) {
	query := `SELECT COUNT(*) FROM people`

	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, errors.NewInternalServerError("Failed to get people count")
	}

	return count, nil
}

func (r *personRepository) Delete(id int) error {
	query := `DELETE FROM people WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return errors.NewInternalServerError("Failed to delete person")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalServerError("Failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("Person not found")
	}

	return nil
}

func (r *personRepository) AddEmail(personID int, email string, isPrimary bool) error {
	query := `INSERT INTO emails (person_id, email, is_primary) VALUES ($1, $2, $3)`

	_, err := r.db.Exec(query, personID, email, isPrimary)
	if err != nil {
		return errors.NewInternalServerError("Failed to add email")
	}

	return nil
}

func (r *personRepository) UpdateEmail(emailID int, email string, isPrimary bool) error {
	query := `UPDATE emails SET email = $1, is_primary = $2 WHERE id = $3`

	result, err := r.db.Exec(query, email, isPrimary, emailID)
	if err != nil {
		return errors.NewInternalServerError("Failed to update email")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalServerError("Failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("Email not found")
	}

	return nil
}

func (r *personRepository) DeleteEmail(emailID int) error {
	query := `DELETE FROM emails WHERE id = $1`

	result, err := r.db.Exec(query, emailID)
	if err != nil {
		return errors.NewInternalServerError("Failed to delete email")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalServerError("Failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("Email not found")
	}

	return nil
}

func (r *personRepository) GetEmails(personID int) ([]models.Email, error) {
	query := `SELECT id, person_id, email, is_primary FROM emails WHERE person_id = $1`

	rows, err := r.db.Query(query, personID)
	if err != nil {
		return nil, errors.NewInternalServerError("Failed to get emails")
	}
	defer rows.Close()

	var emails []models.Email
	for rows.Next() {
		var email models.Email
		err := rows.Scan(&email.ID, &email.PersonID, &email.Email, &email.IsPrimary)
		if err != nil {
			return nil, errors.NewInternalServerError("Failed to scan email")
		}
		emails = append(emails, email)
	}

	return emails, nil
}

func (r *personRepository) AddFriend(personID, friendID int) error {
	query := `INSERT INTO friendships (person_id, friend_id) VALUES ($1, $2)`

	_, err := r.db.Exec(query, personID, friendID)
	if err != nil {
		return errors.NewInternalServerError("Failed to add friend")
	}

	return nil
}

func (r *personRepository) RemoveFriend(personID, friendID int) error {
	query := `DELETE FROM friendships WHERE person_id = $1 AND friend_id = $2`

	result, err := r.db.Exec(query, personID, friendID)
	if err != nil {
		return errors.NewInternalServerError("Failed to remove friend")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalServerError("Failed to get rows affected")
	}

	if rowsAffected == 0 {
		return errors.NewNotFoundError("Friendship not found")
	}

	return nil
}

func (r *personRepository) GetFriends(personID int) ([]models.Person, error) {
	query := `
		SELECT p.id, p.first_name, p.last_name, p.middle_name, p.age, p.gender, p.nationality, p.created_at, p.updated_at
		FROM people p
		JOIN friendships f ON p.id = f.friend_id
		WHERE f.person_id = $1`

	rows, err := r.db.Query(query, personID)
	if err != nil {
		return nil, errors.NewInternalServerError("Failed to get friends")
	}
	defer rows.Close()

	var friends []models.Person
	for rows.Next() {
		person := models.Person{}
		err := rows.Scan(
			&person.ID, &person.FirstName, &person.LastName, &person.MiddleName,
			&person.Age, &person.Gender, &person.Nationality, &person.CreatedAt, &person.UpdatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalServerError("Failed to scan friend")
		}
		friends = append(friends, person)
	}

	return friends, nil
}
