package routes

import (
	"PeopleCRUD/internal/api/handlers"
	"PeopleCRUD/internal/api/middleware"
	"PeopleCRUD/internal/service"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(router *gin.Engine, personService service.PersonService, logger *logrus.Logger) {
	// Middleware
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())
	router.Use(middleware.Timeout(30 * time.Second))

	// Handlers
	peopleHandler := handlers.NewPeopleHandler(personService, logger)

	// Health check
	//router.GET("/health", peopleHandler.HealthCheck)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	v1 := router.Group("/api/v1")
	{
		// People routes
		people := v1.Group("/people")
		{
			people.POST("", peopleHandler.CreatePerson)
			//people.GET("", peopleHandler.GetAllPeople)
			//people.GET("/:id", peopleHandler.GetPerson)
			people.PUT("/:id", peopleHandler.UpdatePerson)
			//people.DELETE("/:id", peopleHandler.DeletePerson)
			//people.GET("/lastname/:lastname", peopleHandler.GetPeopleByLastName)

			// Email routes
			//people.POST("/:id/emails", peopleHandler.AddEmail)

			// Friend routes
			//people.POST("/:id/friends", peopleHandler.AddFriend)
			//people.GET("/:id/friends", peopleHandler.GetFriends)
			//people.DELETE("/:id/friends/:friendId", peopleHandler.RemoveFriend)
		}

		// Email routes
		//emails := v1.Group("/emails")
		//{
		//	emails.PUT("/:emailId", peopleHandler.UpdateEmail)
		//	emails.DELETE("/:emailId", peopleHandler.DeleteEmail)
		//}
	}
}
