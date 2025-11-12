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
 *
 * @return []string
 */
func GetNonCapacityDepartureTerminals() []string {
	departureTerminals := [46]string{
		"TSA", "HSB", "SWB", "NAN", "DUK",
		"NAH", "CMX", "PPH", "BTW", "BKY",
		"CAM", "CHM", "CFT", "MIL", "MCN",
		"LNG", "PWR", "SLT", "ERL", "TEX",
		"POB", "PSB", "PVB", "PST", "GAB",
		"PEN", "PLH", "VES", "FUL", "THT",
		"ALR", "DNM", "DNE", "HRN", "SOI",
		"HRB", "QDR", "BEC", "PBB", "POF",
		"SHW", "KLE", "PPR", "PSK", "ALF", "BOW",
	}

	return departureTerminals[:]
}

/*
 * GetNonCapacityDestinationTerminals
 *
 * Returns an array of destination terminals for non-capacity routes
 *
 * @return [][]string
 */
func GetNonCapacityDestinationTerminals() [][]string {
	destinationTerminals := [46][]string{
		{"PSB", "PVB", "DUK", "POB", "PLH", "PST", "SWB"},
		{"BOW", "NAN", "LNG"},
		{"PSB", "PVB", "POB", "FUL", "PST", "TSA", "PSB", "PVB", "POB", "FUL", "PST"},
		{"HSB"},
		{"TSA"},
		{"GAB"},
		{"PWR"},
		{"PBB", "BEC", "KLE", "POF", "PPR", "SHW", "PBB", "BEC", "KLE", "POF", "PPR", "SHW"},
		{"MIL"},
		{"DNM"},
		{"QDR"},
		{"PEN", "THT", "PEN", "THT"},
		{"VES"},
		{"BTW"},
		{"ALR", "SOI"},
		{"HSB"},
		{"CMX", "TEX"},
		{"ERL"},
		{"SLT"},
		{"PWR"},
		{"PSB", "PVB", "PLH", "PST", "TSA", "SWB"},
		{"PVB", "POB", "PLH", "PST", "TSA", "SWB"},
		{"PSB", "POB", "PLH", "PST", "TSA", "SWB"},
		{"PSB", "PVB", "POB", "PLH", "TSA", "SWB"},
		{"NAH"},
		{"CHM", "THT"},
		{"PSB", "PVB", "POB", "PST", "TSA", "SWB"},
		{"CFT"},
		{"SWB"},
		{"CHM", "PEN"},
		{"SOI", "MCN"},
		{"BKY"},
		{"HRN"},
		{"DNE"},
		{"ALR", "MCN"},
		{"COR"},
		{"CAM"},
		{"PBB", "POF", "PPH", "SHW"},
		{"BEC", "KLE", "POF", "PPH", "PPR", "SHW"},
		{"PBB", "BEC", "PPH", "SHW"},
		{"PBB", "BEC", "POF", "PPH"},
		{"PBB", "PPH", "PPR", "PBB", "PPH"},
		{"PBB", "PSK", "KLE", "PPH", "PSK"},
		{"ALF", "PPR"},
		{"PSK"},
		{"HSB"},
	}

	return destinationTerminals[:]
}
