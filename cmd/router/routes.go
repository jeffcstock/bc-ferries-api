package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/jeffcstock/bc-ferries-api/cmd/db"
	"github.com/jeffcstock/bc-ferries-api/cmd/models"
	"github.com/jeffcstock/bc-ferries-api/cmd/scraper"
	"github.com/jeffcstock/bc-ferries-api/cmd/staticdata"
)

/**************/
/* V2 Structs */
/**************/

type AllDataResponse struct {
	CapacityRoutes    []models.CapacityRoute    `json:"capacityRoutes"`
	NonCapacityRoutes []models.NonCapacityRoute `json:"nonCapacityRoutes"`
}

type CapacityResponse struct {
	Routes []models.CapacityRoute `json:"routes"`
}

type RawSighting struct {
	Terminal     string `json:"terminal"`     // Terminal code (e.g., "TSA")
	TerminalName string `json:"terminalName"` // Full terminal name
	Time         string `json:"time"`         // 12-hour format from scraper (e.g., "7:00 am")
	Time24h      string `json:"time24h"`      // 24-hour format (e.g., "07:00")
	SourceURL    string `json:"sourceUrl"`    // BC Ferries URL where this was scraped
}

type VesselRouteResponse struct {
	VesselName   string                   `json:"vesselName"`
	MMSI         *int                     `json:"mmsi"`        // nullable if not found in mapping
	Date         string                   `json:"date"`        // ISO 8601 date "2025-11-17"
	Movements    []scraper.VesselMovement `json:"movements"`   // Inferred route
	RawSightings []RawSighting            `json:"rawSightings"` // Raw data points from scraper
}

/*************/
/* V2 Routes */
/*************/

/*
 * GetCapacityAndNonCapacitySailings
 *
 * Returns data for all capacity and non capacity routes
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetCapacityAndNonCapacitySailings(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	capacityRoute := db.GetCapacitySailings()
	nonCapacityRoute := db.GetNonCapacitySailings()

	response := AllDataResponse{
		CapacityRoutes:    capacityRoute,
		NonCapacityRoutes: nonCapacityRoute,
	}

	jsonString, _ := json.Marshal(response)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)

}

/*
 * GetCapacitySailings
 *
 * Returns sailing data for all capacity routes
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetCapacitySailings(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	routes := db.GetCapacitySailings()

	response := CapacityResponse{
		Routes: routes,
	}

	if len(response.Routes[0].Sailings) == 0 {
		jsonString, _ := json.Marshal("BC Ferries Data Currently Down")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonString)
	} else {
		jsonString, _ := json.Marshal(response)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonString)
	}
}

/*
 * GetSingleCapacityRoute
 *
 * Returns sailing data for a specific capacity route by route code
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetSingleCapacityRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	routeCode := ps.ByName("routeCode")
	routes := db.GetCapacitySailings()

	// Find the route matching the route code
	var foundRoute *models.CapacityRoute
	for _, route := range routes {
		if route.RouteCode == routeCode {
			foundRoute = &route
			break
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if foundRoute != nil {
		jsonString, _ := json.Marshal(foundRoute)
		w.Write(jsonString)
	} else {
		w.WriteHeader(http.StatusNotFound)
		jsonString, _ := json.Marshal(map[string]string{"error": "Route not found"})
		w.Write(jsonString)
	}
}

/*
 * GetNonCapacitySailings
 *
 * Returns sailing data for all non capacity routes
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetNonCapacitySailings(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	routes := db.GetNonCapacitySailings()

	response := models.NonCapacityResponse{
		Routes: routes,
	}

	jsonString, _ := json.Marshal(response)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)

}

/*
 * GetSingleNonCapacityRoute
 *
 * Returns sailing data for a specific non-capacity route by route code
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetSingleNonCapacityRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	routeCode := ps.ByName("routeCode")
	routes := db.GetNonCapacitySailings()

	// Find the route matching the route code
	var foundRoute *models.NonCapacityRoute
	for _, route := range routes {
		if route.RouteCode == routeCode {
			foundRoute = &route
			break
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if foundRoute != nil {
		jsonString, _ := json.Marshal(foundRoute)
		w.Write(jsonString)
	} else {
		w.WriteHeader(http.StatusNotFound)
		jsonString, _ := json.Marshal(map[string]string{"error": "Route not found"})
		w.Write(jsonString)
	}
}

/*
 * GetCapacityRoutesList
 *
 * Returns lightweight metadata for capacity routes (without sailings).
 * Optionally filters by route codes via query parameter.
 *
 * Query params:
 *   - routeCodes: comma-separated route codes (e.g., "TSASWB,SWBTSA")
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetCapacityRoutesList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Parse query parameter
	routeCodesParam := r.URL.Query().Get("routeCodes")
	var routeCodes []string

	if routeCodesParam != "" {
		routeCodes = strings.Split(routeCodesParam, ",")
		// Trim whitespace from each code
		for i := range routeCodes {
			routeCodes[i] = strings.TrimSpace(routeCodes[i])
		}
	}

	routes := db.GetCapacityRoutesInfo(routeCodes)

	response := models.CapacityRoutesResponse{
		Routes: routes,
	}

	jsonString, _ := json.Marshal(response)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)
}

/*
 * GetNonCapacityRoutesList
 *
 * Returns lightweight metadata for non-capacity routes (without sailings).
 * Optionally filters by route codes via query parameter.
 *
 * Query params:
 *   - routeCodes: comma-separated route codes (e.g., "FULSWB,BOWHSB")
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetNonCapacityRoutesList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Parse query parameter
	routeCodesParam := r.URL.Query().Get("routeCodes")
	var routeCodes []string

	if routeCodesParam != "" {
		routeCodes = strings.Split(routeCodesParam, ",")
		// Trim whitespace from each code
		for i := range routeCodes {
			routeCodes[i] = strings.TrimSpace(routeCodes[i])
		}
	}

	routes := db.GetNonCapacityRoutesInfo(routeCodes)

	response := models.NonCapacityRoutesResponse{
		Routes: routes,
	}

	jsonString, _ := json.Marshal(response)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)
}

/**************/
/* V1 Structs */
/**************/

type Response struct {
	Schedule  map[string]map[string]models.Route `json:"schedule"`
	ScrapedAt time.Time                          `json:"scrapedAt"`
}

/*************/
/* V1 Routes */
/*************/
// V1 routes return data in a different format and only contain upcoming sailings for specific routes

/*
 * GetAllSailings
 *
 * Returns all sailing data
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetAllSailings(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	capacityRoute := db.GetCapacitySailings()
	nonCapacityRoute := db.GetNonCapacitySailings()

	response := AllDataResponse{
		CapacityRoutes:    capacityRoute,
		NonCapacityRoutes: nonCapacityRoute,
	}

	jsonString, _ := json.Marshal(ConvertV1ResponseToV2Response(response))

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)
}

/*
 * GetSailingsByDepartureTerminal
 *
 * Returns sailing data for given departure
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetSailingsByDepartureTerminal(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	departureTerminal := ps.ByName("departureTerminal")
	capacityRoute := db.GetCapacitySailings()
	nonCapacityRoute := db.GetNonCapacitySailings()

	allDataResponse := AllDataResponse{
		CapacityRoutes:    capacityRoute,
		NonCapacityRoutes: nonCapacityRoute,
	}

	jsonString, _ := json.Marshal(ConvertV1ResponseToV2Response(allDataResponse)[departureTerminal])

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)
}

/*
 * GetSailingsByDepartureAndDestinationTerminals
 *
 * Returns sailing data for given departure and destination terminal
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetSailingsByDepartureAndDestinationTerminals(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	departureTerminal := ps.ByName("departureTerminal")
	destinationTerminal := ps.ByName("destinationTerminal")
	capacityRoute := db.GetCapacitySailings()
	nonCapacityRoute := db.GetNonCapacitySailings()

	allDataResponse := AllDataResponse{
		CapacityRoutes:    capacityRoute,
		NonCapacityRoutes: nonCapacityRoute,
	}

	jsonString, _ := json.Marshal(ConvertV1ResponseToV2Response(allDataResponse)[departureTerminal][destinationTerminal])

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)
}

/****************/
/* Other Routes */
/****************/

/*
 * HealthCheck
 *
 * Returns a simple response indicating the server is running.
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func HealthCheck(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	jsonString, _ := json.Marshal("Server OK")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonString)
}

/*
 * GetVesselDatabaseDebug
 *
 * Returns the complete vessel database for debugging purposes.
 * Shows all vessels, terminals, and departure times scraped from BC Ferries.
 *
 * Response format:
 * {
 *   "scrapedAt": "2025-11-23T10:30:00Z",
 *   "terminals": {
 *     "TSA": {
 *       "terminalName": "Tsawwassen",
 *       "departures": [
 *         {"time": "7:00 am", "time24h": "07:00", "vesselName": "Salish Eagle"},
 *         ...
 *       ]
 *     },
 *     ...
 *   },
 *   "vessels": {
 *     "Salish Eagle": {
 *       "sightings": [
 *         {"terminal": "TSA", "terminalName": "Tsawwassen", "time": "7:00 am", "time24h": "07:00"},
 *         ...
 *       ]
 *     },
 *     ...
 *   }
 * }
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetVesselDatabaseDebug(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	vesselDB := scraper.GetVesselDatabase()

	if vesselDB == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Vessel database not yet available"})
		return
	}

	terminals := staticdata.GetTerminals()

	// Build response organized by terminals and vessels
	type Departure struct {
		Time       string `json:"time"`
		Time24h    string `json:"time24h"`
		VesselName string `json:"vesselName"`
	}

	type TerminalDebugInfo struct {
		TerminalName string      `json:"terminalName"`
		TerminalCode string      `json:"terminalCode"`
		Departures   []Departure `json:"departures"`
	}

	type VesselSighting struct {
		Terminal     string `json:"terminal"`
		TerminalName string `json:"terminalName"`
		Time         string `json:"time"`
		Time24h      string `json:"time24h"`
	}

	type VesselDebugInfo struct {
		MMSI      *int             `json:"mmsi"`
		Sightings []VesselSighting `json:"sightings"`
	}

	response := map[string]interface{}{
		"scrapedAt": time.Now().Format(time.RFC3339),
		"terminals": make(map[string]TerminalDebugInfo),
		"vessels":   make(map[string]VesselDebugInfo),
	}

	terminalsMap := response["terminals"].(map[string]TerminalDebugInfo)
	vesselsMap := response["vessels"].(map[string]VesselDebugInfo)

	// Populate terminals view
	for terminalCode, timeMap := range vesselDB {
		terminalName := terminalCode
		if terminal, ok := terminals[terminalCode]; ok {
			terminalName = terminal.Name
		}

		departures := []Departure{}
		for departureTime, vesselName := range timeMap {
			departures = append(departures, Departure{
				Time:       departureTime,
				Time24h:    convertTo24HourFormatHelper(departureTime),
				VesselName: vesselName,
			})
		}

		// Sort departures by time
		sort.Slice(departures, func(i, j int) bool {
			return departures[i].Time24h < departures[j].Time24h
		})

		terminalsMap[terminalCode] = TerminalDebugInfo{
			TerminalName: terminalName,
			TerminalCode: terminalCode,
			Departures:   departures,
		}
	}

	// Populate vessels view
	for terminalCode, timeMap := range vesselDB {
		terminalName := terminalCode
		if terminal, ok := terminals[terminalCode]; ok {
			terminalName = terminal.Name
		}

		for departureTime, vesselName := range timeMap {
			if _, exists := vesselsMap[vesselName]; !exists {
				vesselsMap[vesselName] = VesselDebugInfo{
					MMSI:      staticdata.GetVesselMMSI(vesselName),
					Sightings: []VesselSighting{},
				}
			}

			vesselInfo := vesselsMap[vesselName]
			vesselInfo.Sightings = append(vesselInfo.Sightings, VesselSighting{
				Terminal:     terminalCode,
				TerminalName: terminalName,
				Time:         departureTime,
				Time24h:      convertTo24HourFormatHelper(departureTime),
			})
			vesselsMap[vesselName] = vesselInfo
		}
	}

	// Sort sightings for each vessel by time
	for vesselName, vesselInfo := range vesselsMap {
		sort.Slice(vesselInfo.Sightings, func(i, j int) bool {
			return vesselInfo.Sightings[i].Time24h < vesselInfo.Sightings[j].Time24h
		})
		vesselsMap[vesselName] = vesselInfo
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

/*
 * GetVesselRoute
 *
 * Returns a vessel's complete daily route by reconstructing movements from vessel database.
 * Supports lookup by MMSI (numeric) or vessel slug (kebab-case).
 *
 * @param http.ResponseWriter w
 * @param *http.Request r
 * @param httprouter.Params ps
 *
 * @return void
 */
func GetVesselRoute(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	vesselID := ps.ByName("id")

	// URL decode
	vesselID, err := url.QueryUnescape(vesselID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid vessel ID"})
		return
	}

	var vesselName string

	// Try to parse as MMSI (integer)
	if mmsi, err := strconv.Atoi(vesselID); err == nil {
		// It's an MMSI - lookup vessel name
		vesselName = staticdata.GetVesselNameByMMSI(mmsi)
		if vesselName == "" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "MMSI not found"})
			return
		}
	} else {
		// It's a slug - normalize to proper vessel name
		vesselName = normalizeSlugToVesselName(vesselID)
	}

	// Get vessel database
	vesselDB := scraper.GetVesselDatabase()

	// Check if vessel database is available
	if vesselDB == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "Vessel database not yet available"})
		return
	}

	// Reconstruct vessel route
	movements := scraper.ReconstructVesselRoute(vesselName, vesselDB)

	// Extract raw sightings
	rawSightings := extractRawSightings(vesselName, vesselDB)

	// Check if vessel exists
	if len(movements) == 0 {
		// Check if vessel name is valid (exists in MMSI map)
		mmsi := staticdata.GetVesselMMSI(vesselName)
		if mmsi == nil {
			// Invalid vessel name
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Vessel not found"})
			return
		}
		// Valid vessel but no movements today - return empty array
	}

	// Build response
	response := VesselRouteResponse{
		VesselName:   vesselName,
		MMSI:         staticdata.GetVesselMMSI(vesselName),
		Date:         time.Now().Format("2006-01-02"), // Current date in ISO 8601
		Movements:    movements,
		RawSightings: rawSightings,
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

/********************/
/* Helper Functions */
/********************/

/*
 * ConvertV1ResponseToV2Response
 *
 * Converts the V2 API response format into the legacy V1 structure,
 * organizing sailings by departure and destination terminals.
 *
 * Filters only allowed terminal pairs as defined by internal maps.
 *
 * @param AllDataResponse allData - the combined capacity and non-capacity data
 *
 * @return map[string]map[string]models.Route - nested route data
 */
func ConvertV1ResponseToV2Response(allData AllDataResponse) map[string]map[string]models.Route {
	schedule := make(map[string]map[string]models.Route)

	// Define the allowed terminal codes for CapacityRoutes and NonCapacityRoutes
	capacityRoutesFilter := map[string][]string{
		"TSA": {"SWB", "SGI", "DUK"},
		"SWB": {"TSA", "FUL", "SGI"},
		"HSB": {"NAN", "LNG", "BOW"},
		"DUK": {"TSA"},
		"LNG": {"HSB"},
		"NAN": {"HSB"},
	}

	nonCapacityRoutesFilter := map[string][]string{
		"FUL": {"SWB"},
		"BOW": {"HSB"},
	}

	for _, capRoute := range allData.CapacityRoutes {
		fromTerminal := capRoute.FromTerminalCode
		toTerminal := capRoute.ToTerminalCode

		if allowedDestinations, ok := capacityRoutesFilter[fromTerminal]; ok {
			if contains(allowedDestinations, toTerminal) {
				route := models.Route{
					SailingDuration: capRoute.SailingDuration,
					Sailings:        []models.Sailing{},
				}

                for _, capSailing := range capRoute.Sailings {
                    if capSailing.SailingStatus == "future" || capSailing.SailingStatus == "cancelled" {
                        route.Sailings = append(route.Sailings, models.Sailing{
                            DepartureTime: capSailing.DepartureTime,
                            ArrivalTime:   capSailing.ArrivalTime,
                            IsCancelled:   capSailing.SailingStatus == "cancelled",
                            Fill:          capSailing.Fill,
                            CarFill:       capSailing.CarFill,
                            OversizeFill:  capSailing.OversizeFill,
                            VesselName:    capSailing.VesselName,
                            VesselStatus:  capSailing.VesselStatus,
                        })
                    }
                }

				if len(route.Sailings) > 0 {
					if _, ok := schedule[fromTerminal]; !ok {
						schedule[fromTerminal] = make(map[string]models.Route)
					}
					schedule[fromTerminal][toTerminal] = route
				}
			}
		}
	}

	for _, nonCapRoute := range allData.NonCapacityRoutes {
		fromTerminal := nonCapRoute.FromTerminalCode
		toTerminal := nonCapRoute.ToTerminalCode

		if allowedDestinations, ok := nonCapacityRoutesFilter[fromTerminal]; ok {
			if contains(allowedDestinations, toTerminal) {
				route := models.Route{
					SailingDuration: nonCapRoute.SailingDuration,
					Sailings:        []models.Sailing{},
				}

				for _, nonCapSailing := range nonCapRoute.Sailings {
					route.Sailings = append(route.Sailings, models.Sailing{
						DepartureTime: nonCapSailing.DepartureTime,
						ArrivalTime:   nonCapSailing.ArrivalTime,
						IsCancelled:   false,
						Fill:          0,
						CarFill:       0,
						OversizeFill:  0,
						VesselName:    "",
						VesselStatus:  "",
					})
				}

				if len(route.Sailings) > 0 {
					if _, ok := schedule[fromTerminal]; !ok {
						schedule[fromTerminal] = make(map[string]models.Route)
					}
					schedule[fromTerminal][toTerminal] = route
				}
			}
		}
	}

	return schedule
}

/*
 * contains
 *
 * Utility function to check if a string slice contains a given string.
 *
 * @param []string s - the slice to search
 * @param string str - the string to look for
 *
 * @return bool - true if str is found in s
 */
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

/*
 * normalizeSlugToVesselName
 *
 * Converts a kebab-case vessel slug to Title Case vessel name.
 * Example: "salish-eagle" → "Salish Eagle"
 *
 * @param slug string - Vessel slug in kebab-case
 *
 * @return string - Vessel name in Title Case
 */
func normalizeSlugToVesselName(slug string) string {
	// Split by hyphens
	words := strings.Split(slug, "-")

	// Title case each word
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	// Join with spaces
	return strings.Join(words, " ")
}

/*
 * extractRawSightings
 *
 * Extracts all raw sighting data points for a given vessel from the vessel database.
 * Returns sightings sorted chronologically.
 *
 * @param vesselName string - The vessel name to search for
 * @param vesselDB map[string]map[string]string - The vessel database (terminal → time → vessel)
 *
 * @return []RawSighting - Array of raw sightings sorted by time
 */
func extractRawSightings(vesselName string, vesselDB map[string]map[string]string) []RawSighting {
	var sightings []RawSighting
	vesselNameLower := strings.ToLower(vesselName)
	terminals := staticdata.GetTerminals()

	// Iterate through all terminals and times
	for terminalCode, timeMap := range vesselDB {
		for departureTime, vessel := range timeMap {
			if strings.ToLower(vessel) == vesselNameLower {
				// Get full terminal name
				terminalName := terminalCode
				if terminal, ok := terminals[terminalCode]; ok {
					terminalName = terminal.Name
				}

				// Convert to 24-hour format
				time24h := convertTo24HourFormatHelper(departureTime)

				sighting := RawSighting{
					Terminal:     terminalCode,
					TerminalName: terminalName,
					Time:         departureTime,
					Time24h:      time24h,
					SourceURL:    fmt.Sprintf("https://www.bcferries.com/current-conditions/departures?terminalCode=%s", terminalCode),
				}
				sightings = append(sightings, sighting)
			}
		}
	}

	// Sort by time (24-hour format makes sorting easy)
	sort.Slice(sightings, func(i, j int) bool {
		return sightings[i].Time24h < sightings[j].Time24h
	})

	return sightings
}

/*
 * convertTo24HourFormatHelper
 *
 * Helper function to convert 12-hour time to 24-hour format for sorting.
 * Returns HH:MM format.
 *
 * @param time12h string - Time in 12-hour format (e.g., "7:00 am")
 *
 * @return string - Time in 24-hour format (e.g., "07:00")
 */
func convertTo24HourFormatHelper(time12h string) string {
	layouts := []string{"3:04 pm", "3:04 PM", "03:04 pm", "03:04 PM", "3:04pm", "3:04PM"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, time12h); err == nil {
			return t.Format("15:04")
		}
	}
	return time12h // Return as-is if parsing fails
}
