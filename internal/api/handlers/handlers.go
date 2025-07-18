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

func (h *PeopleHandler) CreatePerson(c *gin.Context) {
	var req models.CreatePersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err.Error()))
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	ctx := c.Request.Context()
	person, err := h.service.CreatePerson(ctx, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, person)
}

func (h *PeopleHandler) GetPerson(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithField("id", c.Param("id")).Warn("Invalid person ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	ctx := c.Request.Context()
	person, err := h.service.GetPersonByID(ctx, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, person)
}

func (h *PeopleHandler) UpdatePerson(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithField("id", c.Param("id")).Warn("Invalid person ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	var req models.UpdatePersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err.Error()))
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	ctx := c.Request.Context()
	person, err := h.service.UpdatePerson(ctx, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, person)
}

func (h *PeopleHandler) DeletePerson(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithField("id", c.Param("id")).Warn("Invalid person ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	ctx := c.Request.Context()
	if err := h.service.DeletePerson(ctx, id); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *PeopleHandler) GetAllPeople(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	ctx := c.Request.Context()
	people, total, err := h.service.GetAllPeople(ctx, limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   people,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *PeopleHandler) AddEmail(c *gin.Context) {
	personID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithField("id", c.Param("id")).Warn("Invalid person ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	var req models.AddEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, errors.NewValidationError(err.Error()))
		return
	}

	ctx := c.Request.Context()
	if err := h.service.AddEmail(ctx, personID, req.Email, req.IsPrimary); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *PeopleHandler) GetFriends(c *gin.Context) {
	personID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithField("id", c.Param("id")).Warn("Invalid person ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	ctx := c.Request.Context()
	friends, err := h.service.GetFriends(ctx, personID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, friends)
}

func (h *PeopleHandler) handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*errors.AppError); ok {
		h.logger.WithFields(logrus.Fields{
			"error": appErr.Error(),
			"code":  appErr.Code,
		}).Error("Handler error")
		c.JSON(appErr.Code, appErr)
	} else {
		h.logger.WithError(err).Error("Unexpected handler error")
		c.JSON(http.StatusInternalServerError, errors.NewInternalServerError("Internal server error"))
	}
}

func (h *PeopleHandler) AddFriend(c *gin.Context) {
	personID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithField("id", c.Param("id")).Warn("Invalid person ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	friendID, err := strconv.Atoi(c.Param("friendId"))
	if err != nil {
		h.logger.WithField("friendId", c.Param("friendId")).Warn("Invalid friend ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid friend ID"))
		return
	}

	ctx := c.Request.Context()
	if err := h.service.AddFriend(ctx, personID, friendID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

func (h *PeopleHandler) RemoveFriend(c *gin.Context) {
	personID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithField("id", c.Param("id")).Warn("Invalid person ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid person ID"))
		return
	}

	friendID, err := strconv.Atoi(c.Param("friendId"))
	if err != nil {
		h.logger.WithField("friendId", c.Param("friendId")).Warn("Invalid friend ID format")
		c.JSON(http.StatusBadRequest, errors.NewValidationError("Invalid friend ID"))
		return
	}

	ctx := c.Request.Context()
	if err := h.service.RemoveFriend(ctx, personID, friendID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
