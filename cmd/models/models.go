package models

import "github.com/jeffcstock/bc-ferries-api/cmd/staticdata"

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
	VesselName         string         `json:"vesselName"`
	VesselStatus       string         `json:"vesselStatus"`
	Events             []SailingEvent `json:"events,omitempty"`
	Legs               []Leg          `json:"legs,omitempty"`
	TotalTravelMin     int            `json:"total_travel_min"`              // Sum of leg sailing durations
	TotalDwellMin      int            `json:"total_dwell_min"`               // Time spent at stops/terminals
	StopCount          int            `json:"stop_count"`                    // Number of stops/transfers (excludes thruFares)
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
	EventType           *string              `json:"event_type"`      // null for final leg, populated for intermediate legs
	DistanceKm          *float64             `json:"distance_km"`     // null if not available
	AvgDurationMin      *int                 `json:"avg_duration_min"` // null if not available
}

/*
 * BuildLegs
 *
 * Constructs sailing legs from route code and events
 * Route code format: OODDDD (first 3 chars = origin, last 3 = destination)
 *
 * @param routeCode string - e.g., "TSAPOB"
 * @param events []SailingEvent - stops, transfers, thru fares
 * @return []Leg - array of leg segments
 */
func BuildLegs(routeCode string, events []SailingEvent) []Leg {
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
			EventType:           nil,
		}

		// Lookup distance and duration
		if legInfo := staticdata.GetLegInfo(originTerminal.Code, destinationTerminal.Code); legInfo != nil {
			leg.DistanceKm = &legInfo.DistanceKm
			leg.AvgDurationMin = &legInfo.AvgDurationMin
		}

		legs = append(legs, leg)
		return legs
	}

	// Build legs from events
	currentOrigin := originTerminal

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
			EventType:           &event.Type,
		}

		// Lookup distance and duration
		if legInfo := staticdata.GetLegInfo(currentOrigin.Code, eventTerminal.Code); legInfo != nil {
			leg.DistanceKm = &legInfo.DistanceKm
			leg.AvgDurationMin = &legInfo.AvgDurationMin
		}

		legs = append(legs, leg)
		currentOrigin = eventTerminal
	}

	// Add final leg from last event to destination
	finalLeg := Leg{
		LegNumber:           len(legs) + 1,
		OriginTerminal:      currentOrigin,
		DestinationTerminal: destinationTerminal,
		EventType:           nil,
	}

	// Lookup distance and duration
	if legInfo := staticdata.GetLegInfo(currentOrigin.Code, destinationTerminal.Code); legInfo != nil {
		finalLeg.DistanceKm = &legInfo.DistanceKm
		finalLeg.AvgDurationMin = &legInfo.AvgDurationMin
	}

	legs = append(legs, finalLeg)
	return legs
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
