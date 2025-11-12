package router

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

/*
 * SetupRouter
 *
 * Initializes the HTTP router and registers all API endpoints.
 * Also serves static files for not-found routes.
 *
 * @return *httprouter.Router - configured router instance
 */
func SetupRouter() *httprouter.Router {
	router := httprouter.New()

	// V2 Routes (with and without trailing slash)
	router.GET("/v2", GetCapacityAndNonCapacitySailings)
	router.GET("/v2/", GetCapacityAndNonCapacitySailings)

	// Routes list endpoints (moved to avoid conflict with :routeCode wildcard)
	router.GET("/v2/routes/capacity", GetCapacityRoutesList)
	router.GET("/v2/routes/capacity/", GetCapacityRoutesList)
	router.GET("/v2/routes/noncapacity", GetNonCapacityRoutesList)
	router.GET("/v2/routes/noncapacity/", GetNonCapacityRoutesList)

	// Capacity routes
	router.GET("/v2/capacity", GetCapacitySailings)
	router.GET("/v2/capacity/", GetCapacitySailings)
	router.GET("/v2/capacity/:routeCode", GetSingleCapacityRoute)
	router.GET("/v2/capacity/:routeCode/", GetSingleCapacityRoute)

	// Non-capacity routes
	router.GET("/v2/noncapacity", GetNonCapacitySailings)
	router.GET("/v2/noncapacity/", GetNonCapacitySailings)
	router.GET("/v2/noncapacity/:routeCode", GetSingleNonCapacityRoute)
	router.GET("/v2/noncapacity/:routeCode/", GetSingleNonCapacityRoute)

	// V1 Routes (with and without trailing slash)
	router.GET("/api", GetAllSailings)
	router.GET("/api/", GetAllSailings)
	router.GET("/api/:departureTerminal", GetSailingsByDepartureTerminal)
	router.GET("/api/:departureTerminal/", GetSailingsByDepartureTerminal)
	router.GET("/api/:departureTerminal/:destinationTerminal", GetSailingsByDepartureAndDestinationTerminals)
	router.GET("/api/:departureTerminal/:destinationTerminal/", GetSailingsByDepartureAndDestinationTerminals)

	router.GET("/healthcheck", HealthCheck)
	router.GET("/healthcheck/", HealthCheck)

	router.NotFound = http.FileServer(http.Dir("./static"))

	return router
}
