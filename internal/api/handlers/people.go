package handlers

import (
	"PeopleCRUD/internal/models"
	"PeopleCRUD/internal/service"
	"PeopleCRUD/pkg/errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PeopleHandler struct {
	service service.PersonService
	logger  *logrus.Logger
}

func NewPeopleHandler(service service.PersonService, logger *logrus.Logger) *PeopleHandler {
	return &PeopleHandler{
		service: service,
		logger:  logger,
	}
}

func (h *PeopleHandler) UpdatePerson(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	var req models.UpdatePersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err.Error()))
		return
	}

	// Валидация запроса
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	// Проверка контекста
	ctx := c.Request.Context()
	if ctx.Err() != nil {
		c.JSON(http.StatusRequestTimeout, errors.NewInternalServerError("Request timeout"))
		return
	}

	// Используем возвращаемое значение из UpdatePerson
	person, err := h.service.UpdatePerson(ctx, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, person)
}

func (h *PeopleHandler) CreatePerson(c *gin.Context) {
	var req models.CreatePersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err.Error()))
		return
	}

	// Валидация запроса
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	ctx := c.Request.Context()
	if ctx.Err() != nil {
		c.JSON(http.StatusRequestTimeout, errors.NewInternalServerError("Request timeout"))
		return
	}

	person, err := h.service.CreatePerson(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, person)
}

func (h *PeopleHandler) handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		c.JSON(appErr.Code, appErr)
	} else {
		h.logger.WithError(err).Error("Unexpected error")
		c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("Internal server error"))
	}
}
