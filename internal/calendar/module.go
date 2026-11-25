package calendar

import (
	"github.com/gin-gonic/gin"
	"go-service/internal/calendar/application"
	"go-service/internal/calendar/infrastructure/acl"
	"go-service/internal/calendar/infrastructure/persistence/postgres"
	"go-service/internal/calendar/interfaces/rest"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
	sharedpostgres "go-service/internal/shared/postgres"
)

// NewModule initializes the calendar module and registers its HTTP routes.
func NewModule(
	db *sharedpostgres.Client,
	router gin.IRouter,
	personResolver application.PersonResolver,
	customerReader *crm.CustomerReader,
	serviceReader *catalog.ServiceReader,
	scheduleReader *organizations.ScheduleReader,
) {
	store := postgres.NewStore(db)
	service := application.NewService(
		store,
		personResolver,
		acl.NewCustomerAdapter(customerReader),
		acl.NewServiceAdapter(serviceReader),
		acl.NewOrganizationAdapter(scheduleReader),
	)
	rest.RegisterRoutes(router.Group("/v1"), service)
}
