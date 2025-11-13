package models

import (
	"strings"
	"time"

	"github.com/jeffcstock/bc-ferries-api/cmd/staticdata"
)

// For shared structs

/**************/
/* V2 Structs */
/**************/

type CapacityRoute struct {
	Date             string            `json:"date"`
	RouteCode        string            `json:"routeCode"`
	FromTerminalCode string            `json:"fromTerminalCode"`
	ToTerminalCode   string            `json:"toTerminalCode"`
	SailingDuration  string            `json:"sailingDuration"`
	Sailings         []CapacitySailing `json:"sailings"`
}

type CapacityRouteInfo struct {
	Date             string `json:"date"`
	RouteCode        string `json:"routeCode"`
	FromTerminalCode string `json:"fromTerminalCode"`
	ToTerminalCode   string `json:"toTerminalCode"`
	SailingDuration  string `json:"sailingDuration"`
}

type CapacitySailing struct {
	ID            string `json:"id"`
	DepartureTime string `json:"time"`
	ArrivalTime   string `json:"arrivalTime"`
	SailingStatus string `json:"sailingStatus"`
	Fill          int    `json:"fill"`
	CarFill       int    `json:"carFill"`
	OversizeFill  int    `json:"oversizeFill"`
	VesselName    string `json:"vesselName"`
	VesselStatus  string `json:"vesselStatus"`
}

type NonCapacityResponse struct {
	Routes []NonCapacityRoute `json:"routes"`
}

type CapacityRoutesResponse struct {
	Routes []CapacityRouteInfo `json:"routes"`
}

type NonCapacityRoutesResponse struct {
	Routes []NonCapacityRouteInfo `json:"routes"`
}

type NonCapacityRoute struct {
	Date             string               `json:"date"`
	RouteCode        string               `json:"routeCode"`
	FromTerminalCode string               `json:"fromTerminalCode"`
	ToTerminalCode   string               `json:"toTerminalCode"`
	SailingDuration  string               `json:"sailingDuration"`
	Sailings         []NonCapacitySailing `json:"sailings"`
}

type NonCapacityRouteInfo struct {
	Date             string `json:"date"`
	RouteCode        string `json:"routeCode"`
	FromTerminalCode string `json:"fromTerminalCode"`
	ToTerminalCode   string `json:"toTerminalCode"`
	SailingDuration  string `json:"sailingDuration"`
}

type NonCapacitySailing struct {
	ID                 string         `json:"id"`
	DepartureTime      string         `json:"time"`
	ArrivalTime        string         `json:"arrivalTime"`
	SailingDuration    string         `json:"sailingDuration"`
	IsNonStop          bool           `json:"isNonStop"`                     // True if direct sailing with no stops/transfers
	HasStops           bool           `json:"hasStops"`                      // True if sailing contains at least one stop event
	IsThruFare         bool           `json:"isThruFare"`                    // True if sailing contains at least one thru-fare event
	Events             []SailingEvent `json:"events,omitempty"`
	Legs               []Leg          `json:"legs,omitempty"`
	TotalTravelMin     int            `json:"total_travel_min"`              // Sum of leg sailing durations
	TotalDwellMin      int            `json:"total_dwell_min"`               // Time spent at stops/terminals
	AvgDwellPerStopMin *int           `json:"avg_dwell_per_stop_min,omitempty"` // Average dwell time per stop
}

type SailingEvent struct {
	Type         string `json:"type"`         // "thruFare", "stop", or "transfer"
	TerminalName string `json:"terminalName"` // e.g., "Victoria (Swartz Bay)"
}

type Leg struct {
	LegNumber           int                  `json:"leg_number"`
	OriginTerminal      staticdata.Terminal  `json:"origin_terminal"`
	DestinationTerminal staticdata.Terminal  `json:"destination_terminal"`
	DistanceKm          *float64             `json:"distance_km"`     // null if not available
	AvgDurationMin      *int                 `json:"avg_duration_min"` // null if not available
	VesselName          *string              `json:"vessel_name"`      // null if not available, "UNKNOWN" if lookup failed
}

/*
 * BuildLegs
 *
 * Constructs sailing legs from route code and events, including vessel name lookups
 * Route code format: OODDDD (first 3 chars = origin, last 3 = destination)
 *
 * @param routeCode string - e.g., "TSAPOB"
 * @param events []SailingEvent - stops, transfers, thru fares
 * @param sailingDepartureTime string - Departure time of the sailing (e.g., "7:10 am")
 * @param vesselDatabase map[string]map[string]string - Vessel database (terminal → time → vessel)
 * @param avgDwellMin int - Average dwell time per stop in minutes
 * @return []Leg - array of leg segments
 */
func BuildLegs(routeCode string, events []SailingEvent, sailingDepartureTime string, vesselDatabase map[string]map[string]string, avgDwellMin int) []Leg {
	terminals := staticdata.GetTerminals()

	// Extract origin and destination codes from route code
	originCode := routeCode[:3]
	destinationCode := routeCode[3:]

	// Get origin and destination terminals
	originTerminal, originExists := terminals[originCode]
	destinationTerminal, destExists := terminals[destinationCode]

	// If terminals don't exist, return empty legs
	if !originExists || !destExists {
		return []Leg{}
	}

	var legs []Leg

	// No events = direct sailing
	if len(events) == 0 {
		leg := Leg{
			LegNumber:           1,
			OriginTerminal:      originTerminal,
			DestinationTerminal: destinationTerminal,
		}

		// Lookup distance and duration
		if legInfo := staticdata.GetLegInfo(originTerminal.Code, destinationTerminal.Code); legInfo != nil {
			leg.DistanceKm = &legInfo.DistanceKm
			leg.AvgDurationMin = &legInfo.AvgDurationMin
		}

		// Lookup vessel for first leg
		if terminalDB, ok := vesselDatabase[originTerminal.Code]; ok {
			leg.VesselName = findVesselByTimeWindow(terminalDB, sailingDepartureTime, 60)
		}

		legs = append(legs, leg)
		return legs
	}

	// Build legs from events
	currentOrigin := originTerminal
	elapsedMinutes := 0 // Track cumulative time for subsequent leg lookups

	for _, event := range events {
		// Map event terminal name to code
		eventTerminalCode := staticdata.GetTerminalCodeByName(event.TerminalName)

		var eventTerminal staticdata.Terminal
		if eventTerminalCode == "" {
			// Unknown terminal - create UNKNOWN terminal with original name
			eventTerminal = staticdata.Terminal{
				Code:        "UNKNOWN",
				Name:        event.TerminalName,
				ServiceArea: "UNKNOWN",
				Lat:         0,
				Lon:         0,
			}
		} else {
			var exists bool
			eventTerminal, exists = terminals[eventTerminalCode]
			if !exists {
				// Terminal code not found - create UNKNOWN terminal
				eventTerminal = staticdata.Terminal{
					Code:        "UNKNOWN",
					Name:        event.TerminalName,
					ServiceArea: "UNKNOWN",
					Lat:         0,
					Lon:         0,
				}
			}
		}

		// Create leg to this event terminal
		leg := Leg{
			LegNumber:           len(legs) + 1,
			OriginTerminal:      currentOrigin,
			DestinationTerminal: eventTerminal,
		}

		// Lookup distance and duration
		if legInfo := staticdata.GetLegInfo(currentOrigin.Code, eventTerminal.Code); legInfo != nil {
			leg.DistanceKm = &legInfo.DistanceKm
			leg.AvgDurationMin = &legInfo.AvgDurationMin
		}

		// Lookup vessel name
		if len(legs) == 0 {
			// First leg: use sailing departure time
			if terminalDB, ok := vesselDatabase[currentOrigin.Code]; ok {
				leg.VesselName = findVesselByTimeWindow(terminalDB, sailingDepartureTime, 60)
			}
		} else {
			// Subsequent legs: check previous event type
			prevEvent := events[len(legs)-1]
			if prevEvent.Type == "stop" {
				// Same vessel continues
				leg.VesselName = legs[len(legs)-1].VesselName
			} else if prevEvent.Type == "transfer" || prevEvent.Type == "thruFare" {
				// Different vessel: calculate estimated departure time and lookup
				estimatedTime := calculateEstimatedTime(sailingDepartureTime, elapsedMinutes+avgDwellMin)
				if terminalDB, ok := vesselDatabase[currentOrigin.Code]; ok {
					leg.VesselName = findVesselByTimeWindow(terminalDB, estimatedTime, 60)
				}
			}
		}

		// Update elapsed time for next leg
		if leg.AvgDurationMin != nil {
			elapsedMinutes += *leg.AvgDurationMin
		}

		legs = append(legs, leg)
		currentOrigin = eventTerminal
	}

	// Add final leg from last event to destination
	finalLeg := Leg{
		LegNumber:           len(legs) + 1,
		OriginTerminal:      currentOrigin,
		DestinationTerminal: destinationTerminal,
	}

	// Lookup distance and duration
	if legInfo := staticdata.GetLegInfo(currentOrigin.Code, destinationTerminal.Code); legInfo != nil {
		finalLeg.DistanceKm = &legInfo.DistanceKm
		finalLeg.AvgDurationMin = &legInfo.AvgDurationMin
	}

	// Lookup vessel for final leg
	if len(events) > 0 {
		lastEvent := events[len(events)-1]
		if lastEvent.Type == "stop" {
			// Same vessel continues
			finalLeg.VesselName = legs[len(legs)-1].VesselName
		} else if lastEvent.Type == "transfer" || lastEvent.Type == "thruFare" {
			// Different vessel: calculate estimated departure time and lookup
			estimatedTime := calculateEstimatedTime(sailingDepartureTime, elapsedMinutes+avgDwellMin)
			if terminalDB, ok := vesselDatabase[currentOrigin.Code]; ok {
				finalLeg.VesselName = findVesselByTimeWindow(terminalDB, estimatedTime, 60)
			}
		}
	}

	legs = append(legs, finalLeg)
	return legs
}

/*
 * Helper function to find vessel by time window (internal version for use within models package)
 */
func findVesselByTimeWindow(vesselDB map[string]string, targetTime string, windowMinutes int) *string {
	unknown := "UNKNOWN"

	// This is a simplified version - the full implementation is in scraper package
	// For the models package, we just do a direct lookup first
	if vessel, ok := vesselDB[targetTime]; ok {
		return &vessel
	}

	// If no exact match, return UNKNOWN
	// The scraper package has the full time-window matching logic
	return &unknown
}

/*
 * Helper function to calculate estimated time by adding minutes to a base time
 */
func calculateEstimatedTime(baseTime string, addMinutes int) string {
	// Parse base time
	layouts := []string{"3:04 pm", "3:04 PM", "03:04 pm", "03:04 PM", "3:04pm", "3:04PM"}
	var baseParsed time.Time
	var err error
	for _, layout := range layouts {
		baseParsed, err = time.Parse(layout, baseTime)
		if err == nil {
			break
		}
	}
	if err != nil {
		return baseTime // Fallback to original time if parsing fails
	}

	// Add minutes
	estimatedParsed := baseParsed.Add(time.Duration(addMinutes) * time.Minute)

	// Format back to lowercase 12-hour format
	return strings.ToLower(estimatedParsed.Format("3:04 pm"))
}

/**************/
/* V1 Structs */
/**************/

type Route struct {
	SailingDuration string    `json:"sailingDuration"`
	Sailings        []Sailing `json:"sailings"`
}

type Sailing struct {
	DepartureTime string `json:"time"`
	ArrivalTime   string `json:"arrivalTime"`
	IsCancelled   bool   `json:"isCancelled"`
	Fill          int    `json:"fill"`
	CarFill       int    `json:"carFill"`
	OversizeFill  int    `json:"oversizeFill"`
	VesselName    string `json:"vesselName"`
	VesselStatus  string `json:"vesselStatus"`
}
