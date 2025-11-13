package staticdata

/*
 * Terminal
 *
 * Represents a ferry terminal with its identifying information and geographic coordinates
 */
type Terminal struct {
	Code        string
	Name        string
	ServiceArea string
	Lat         float64
	Lon         float64
}

/*
 * GetTerminals
 *
 * Returns a map of terminal codes to their detailed information
 *
 * @return map[string]Terminal
 */
func GetTerminals() map[string]Terminal {
	return map[string]Terminal{
		"TSA": {
			Code:        "TSA",
			Name:        "Tsawwassen",
			ServiceArea: "Vancouver",
			Lat:         49.00668815099993,
			Lon:         -123.131201540845922,
		},
		"POB": {
			Code:        "POB",
			Name:        "Otter Bay",
			ServiceArea: "Pender Island",
			Lat:         48.80057460745268,
			Lon:         -123.3156620925309,
		},
		"SWB": {
			Code:        "SWB",
			Name:        "Swartz Bay",
			ServiceArea: "Victoria",
			Lat:         48.689514131680525,
			Lon:         -123.41159059969662,
		},
		"PLH": {
			Code:        "PLH",
			Name:        "Long Harbour",
			ServiceArea: "Salt Spring Island",
			Lat:         48.85202048619191,
			Lon:         -123.44573195530967,
		},
		"PVB": {
			Code:        "PVB",
			Name:        "Village Bay",
			ServiceArea: "Mayne Island",
			Lat:         48.84469158068223,
			Lon:         -123.32457711184203,
		},
		"PST": {
			Code:        "PST",
			Name:        "Lyall Harbour",
			ServiceArea: "Saturna Island",
			Lat:         48.7980858638295,
			Lon:         -123.20133068520887,
		},
		"FUL": {
			Code:        "FUL",
			Name:        "Fulford Harbour",
			ServiceArea: "Salt Spring Island",
			Lat:         48.7693511284093,
			Lon:         -123.45103007478454,
		},
		"PSB": {
			Code:        "PSB",
			Name:        "Sturdies Bay",
			ServiceArea: "Galiano Island",
			Lat:         48.876705467392696,
			Lon:         -123.31512438289518,
		},
	}
}

/*
 * GetCapacityDepartureTerminals
 *
 * Returns an array of departure terminals for capacity routes
 *
 * @return []string
 */
func GetCapacityDepartureTerminals() []string {
	departureTerminals := [6]string{
		"TSA",
		"SWB",
		"HSB",
		"DUK",
		"LNG",
		"NAN",
	}

	return departureTerminals[:]
}

/*
 * GetCapacityDestinationTerminals
 *
 * Returns an array of destination terminals for capacity routes
 *
 * @return [][]string
 */
func GetCapacityDestinationTerminals() [][]string {
	destinationTerminals := [6][]string{
		{"SWB", "SGI", "DUK"},
		{"TSA", "FUL", "SGI"},
		{"NAN", "LNG", "BOW"},
		{"TSA"},
		{"HSB"},
		{"HSB"},
	}

	return destinationTerminals[:]
}

/*
 * GetNonCapacityDepartureTerminals
 *
 * Returns an array of departure terminals for non-capacity routes
 * Limited to Southern Gulf Islands routes only to reduce memory usage
 *
 * @return []string
 */
func GetNonCapacityDepartureTerminals() []string {
	// Southern Gulf Islands routes only
	departureTerminals := [8]string{
		"TSA",  // Tsawwassen
		"SWB",  // Swartz Bay
		"POB",  // Otter Bay (Pender Island)
		"PSB",  // Sturdies Bay (Galiano Island)
		"PVB",  // Village Bay (Mayne Island)
		"PST",  // Lyall Harbour (Saturna Island)
		"PLH",  // Long Harbour (Salt Spring Island)
		"FUL",  // Fulford Harbour (Salt Spring Island)
	}

	return departureTerminals[:]
}

/*
 * GetNonCapacityDestinationTerminals
 *
 * Returns an array of destination terminals for non-capacity routes
 * Limited to Southern Gulf Islands routes only to reduce memory usage
 *
 * @return [][]string
 */
func GetNonCapacityDestinationTerminals() [][]string {
	// Southern Gulf Islands destinations only - using EXACT mappings from original working code
	// Order matches GetNonCapacityDepartureTerminals: TSA, SWB, POB, PSB, PVB, PST, PLH, FUL
	destinationTerminals := [8][]string{
		{"PSB", "PVB", "DUK", "POB", "PLH", "PST", "SWB"},                              // From TSA (original index 0)
		{"PSB", "PVB", "POB", "FUL", "PST", "TSA", "PSB", "PVB", "POB", "FUL", "PST"},  // From SWB (original index 2)
		{"PSB", "PVB", "PLH", "PST", "TSA", "SWB"},                                     // From POB (original index 20)
		{"PVB", "POB", "PLH", "PST", "TSA", "SWB"},                                     // From PSB (original index 21)
		{"PSB", "POB", "PLH", "PST", "TSA", "SWB"},                                     // From PVB (original index 22)
		{"PSB", "PVB", "POB", "PLH", "TSA", "SWB"},                                     // From PST (original index 23)
		{"PSB", "PVB", "POB", "PST", "TSA", "SWB"},                                     // From PLH (original index 26)
		{"SWB"},                                                                         // From FUL (original index 28)
	}

	return destinationTerminals[:]
}

/*
 * GetTerminalCodeByName
 *
 * Maps a terminal name in format "ServiceArea (Name)" to its terminal code
 * e.g., "Galiano Island (Sturdies Bay)" -> "PSB"
 *
 * @param terminalName string - Terminal name in format "ServiceArea (Name)"
 * @return string - Terminal code (empty string if not found)
 */
func GetTerminalCodeByName(terminalName string) string {
	terminals := GetTerminals()

	for code, terminal := range terminals {
		// Format: "ServiceArea (Name)"
		expectedFormat := terminal.ServiceArea + " (" + terminal.Name + ")"
		if expectedFormat == terminalName {
			return code
		}
	}

	return "" // Not found
}
