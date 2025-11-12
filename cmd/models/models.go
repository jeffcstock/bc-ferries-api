package models

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
	ID              string         `json:"id"`
	DepartureTime   string         `json:"time"`
	ArrivalTime     string         `json:"arrivalTime"`
	SailingDuration string         `json:"sailingDuration"`
	VesselName      string         `json:"vesselName"`
	VesselStatus    string         `json:"vesselStatus"`
	Events          []SailingEvent `json:"events,omitempty"`
}

type SailingEvent struct {
	Type         string `json:"type"`         // "thruFare", "stop", or "transfer"
	TerminalName string `json:"terminalName"` // e.g., "Victoria (Swartz Bay)"
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
