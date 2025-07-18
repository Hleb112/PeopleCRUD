package routes

import (
	"PeopleCRUD/internal/api/handlers"
	"PeopleCRUD/internal/api/middleware"
	"PeopleCRUD/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func SetupRoutes(router *gin.Engine, personService service.PersonService, logger *logrus.Logger) {
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())
	router.Use(middleware.Timeout(30 * time.Second))

	peopleHandler := handlers.NewPeopleHandler(personService, logger)

	api := router.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			v1.POST("/people", peopleHandler.CreatePerson)
			v1.GET("/people", peopleHandler.GetAllPeople)
			v1.GET("/people/:id", peopleHandler.GetPerson)
			v1.PUT("/people/:id", peopleHandler.UpdatePerson)
			v1.DELETE("/people/:id", peopleHandler.DeletePerson)

			v1.POST("/people/:id/emails", peopleHandler.AddEmail)
			v1.GET("/people/:id/friends", peopleHandler.GetFriends)
		}
	}
}
