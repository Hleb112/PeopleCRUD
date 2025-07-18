package models

import (
	"PeopleCRUD/pkg/errors"
	"regexp"
	"strings"
	"time"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Person struct {
	ID          int       `json:"id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	MiddleName  *string   `json:"middle_name,omitempty"`
	Age         *int      `json:"age,omitempty"`
	Gender      *string   `json:"gender,omitempty"`
	Nationality *string   `json:"nationality,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PersonWithDetails struct {
	Person
	Emails  []Email  `json:"emails,omitempty"`
	Friends []Person `json:"friends,omitempty"`
}

type Email struct {
	ID        int       `json:"id"`
	PersonID  int       `json:"person_id"`
	Email     string    `json:"email"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

type CreatePersonRequest struct {
	FirstName  string   `json:"first_name" binding:"required"`
	LastName   string   `json:"last_name" binding:"required"`
	MiddleName *string  `json:"middle_name,omitempty"`
	Emails     []string `json:"emails,omitempty"`
}

func (r *CreatePersonRequest) Validate() error {
	if strings.TrimSpace(r.FirstName) == "" {
		return errors.NewValidationError("First name is required")
	}
	if strings.TrimSpace(r.LastName) == "" {
		return errors.NewValidationError("Last name is required")
	}
	for _, email := range r.Emails {
		if !emailRegex.MatchString(email) {
			return errors.NewValidationError("Invalid email format: " + email)
		}
	}
	return nil
}

type UpdatePersonRequest struct {
	FirstName   *string `json:"first_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	MiddleName  *string `json:"middle_name,omitempty"`
	Age         *int    `json:"age,omitempty"`
	Gender      *string `json:"gender,omitempty"`
	Nationality *string `json:"nationality,omitempty"`
}

func (r *UpdatePersonRequest) Validate() error {
	if r.FirstName != nil && strings.TrimSpace(*r.FirstName) == "" {
		return errors.NewValidationError("First name cannot be empty")
	}
	if r.LastName != nil && strings.TrimSpace(*r.LastName) == "" {
		return errors.NewValidationError("Last name cannot be empty")
	}
	return nil
}

type AddEmailRequest struct {
	Email     string `json:"email" binding:"required"`
	IsPrimary bool   `json:"is_primary"`
}
