package models

import (
	"PeopleCRUD/pkg/errors"
	"regexp"
	"strings"
	"time"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Person struct {
	ID          int       `json:"id" db:"id"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
	MiddleName  *string   `json:"middle_name,omitempty" db:"middle_name"`
	Age         *int      `json:"age,omitempty" db:"age"`
	Gender      *string   `json:"gender,omitempty" db:"gender"`
	Nationality *string   `json:"nationality,omitempty" db:"nationality"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type PersonWithDetails struct {
	Person
	Emails  []Email  `json:"emails,omitempty"`
	Friends []Person `json:"friends,omitempty"`
}

type Email struct {
	ID        int       `json:"id" db:"id"`
	PersonID  int       `json:"person_id" db:"person_id"`
	Email     string    `json:"email" db:"email"`
	IsPrimary bool      `json:"is_primary" db:"is_primary"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CreatePersonRequest struct {
	FirstName  string   `json:"first_name" binding:"required,min=1,max=100"`
	LastName   string   `json:"last_name" binding:"required,min=1,max=100"`
	MiddleName *string  `json:"middle_name,omitempty" binding:"omitempty,max=100"`
	Emails     []string `json:"emails,omitempty" binding:"dive,email"`
}

type UpdatePersonRequest struct {
	FirstName   *string `json:"first_name,omitempty" binding:"omitempty,min=1,max=100"`
	LastName    *string `json:"last_name,omitempty" binding:"omitempty,min=1,max=100"`
	MiddleName  *string `json:"middle_name,omitempty" binding:"omitempty,max=100"`
	Age         *int    `json:"age,omitempty" binding:"omitempty,min=1,max=150"`
	Gender      *string `json:"gender,omitempty" binding:"omitempty,oneof=male female"`
	Nationality *string `json:"nationality,omitempty" binding:"omitempty,len=2"`
}

func (r *CreatePersonRequest) Validate() error {
	if strings.TrimSpace(r.FirstName) == "" {
		return errors.NewValidationError("First name is required")
	}
	if strings.TrimSpace(r.LastName) == "" {
		return errors.NewValidationError("Last name is required")
	}
	if len(r.FirstName) > 100 {
		return errors.NewValidationError("First name too long")
	}
	if len(r.LastName) > 100 {
		return errors.NewValidationError("Last name too long")
	}

	for _, email := range r.Emails {
		if !emailRegex.MatchString(email) {
			return errors.NewValidationError("Invalid email format: " + email)
		}
	}

	return nil
}

func (r *UpdatePersonRequest) Validate() error {
	if r.FirstName == nil && r.LastName == nil && r.MiddleName == nil &&
		r.Age == nil && r.Gender == nil && r.Nationality == nil {
		return errors.NewValidationError("At least one field must be provided for update")
	}

	if r.FirstName != nil && strings.TrimSpace(*r.FirstName) == "" {
		return errors.NewValidationError("First name cannot be empty")
	}
	if r.LastName != nil && strings.TrimSpace(*r.LastName) == "" {
		return errors.NewValidationError("Last name cannot be empty")
	}
	if r.Age != nil && (*r.Age < 1 || *r.Age > 150) {
		return errors.NewValidationError("Age must be between 1 and 150")
	}
	if r.Gender != nil && (*r.Gender != "male" && *r.Gender != "female") {
		return errors.NewValidationError("Gender must be 'male' or 'female'")
	}
	if r.Nationality != nil && len(*r.Nationality) != 2 {
		return errors.NewValidationError("Nationality must be 2 characters")
	}

	return nil
}
