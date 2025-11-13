package staticdata

/*
 * LegInfo
 *
 * Contains distance and average duration information for a ferry route leg
 */
type LegInfo struct {
	DistanceKm     float64
	AvgDurationMin int
}

/*
 * GetLegInfo
 *
 * Returns distance and duration information for a specific route leg
 * Key format: "ORIGIN-DESTINATION" (e.g., "TSA-PSB")
 *
 * @param originCode string - Origin terminal code
 * @param destinationCode string - Destination terminal code
 * @return *LegInfo - Leg information (nil if not found)
 */
func GetLegInfo(originCode, destinationCode string) *LegInfo {
	key := originCode + "-" + destinationCode
	if info, exists := legData[key]; exists {
		return &info
	}
	return nil
}

/*
 * legData
 *
 * Hardcoded lookup table for route leg distances and average durations
 * Data source: BC Ferries schedule information
 *
 * Note: Not all routes are populated yet. Missing routes will return nil.
 */
var legData = map[string]LegInfo{
	// Southern Gulf Islands routes
	"TSA-PSB": {DistanceKm: 22.5, AvgDurationMin: 55},
	"PSB-TSA": {DistanceKm: 22.5, AvgDurationMin: 55},

	"TSA-PVB": {DistanceKm: 27.1, AvgDurationMin: 70},
	"PVB-TSA": {DistanceKm: 27.1, AvgDurationMin: 70},

	"TSA-POB": {DistanceKm: 35, AvgDurationMin: 80},
	"POB-TSA": {DistanceKm: 35, AvgDurationMin: 85},

	"TSA-PLH": {DistanceKm: 40, AvgDurationMin: 85},
	"PLH-TSA": {DistanceKm: 40, AvgDurationMin: 85},

	"PST-PVB": {DistanceKm: 13, AvgDurationMin: 35},
	"PVB-PST": {DistanceKm: 13, AvgDurationMin: 35},

	"PST-POB": {DistanceKm: 12.4, AvgDurationMin: 40},
	"POB-PST": {DistanceKm: 12.4, AvgDurationMin: 40},

	"PLH-POB": {DistanceKm: 13, AvgDurationMin: 40},
	"POB-PLH": {DistanceKm: 13, AvgDurationMin: 40},

	"PLH-PVB": {DistanceKm: 11.3, AvgDurationMin: 35},
	"PVB-PLH": {DistanceKm: 11.3, AvgDurationMin: 35},

	"POB-PVB": {DistanceKm: 6.9, AvgDurationMin: 25},
	"PVB-POB": {DistanceKm: 6.9, AvgDurationMin: 30},

	"POB-PSB": {DistanceKm: 14.3, AvgDurationMin: 45},
	"PSB-POB": {DistanceKm: 14.3, AvgDurationMin: 40},

	"PVB-PSB": {DistanceKm: 8, AvgDurationMin: 25},
	"PSB-PVB": {DistanceKm: 8, AvgDurationMin: 30},

	// Swartz Bay routes
	"TSA-SWB": {DistanceKm: 46.6, AvgDurationMin: 95},
	"SWB-TSA": {DistanceKm: 46.6, AvgDurationMin: 95},

	"SWB-POB": {DistanceKm: 14, AvgDurationMin: 40},
	"POB-SWB": {DistanceKm: 14, AvgDurationMin: 42},

	"SWB-PVB": {DistanceKm: 21.6, AvgDurationMin: 55},
	"PVB-SWB": {DistanceKm: 21.6, AvgDurationMin: 53},

	"SWB-PSB": {DistanceKm: 27.6, AvgDurationMin: 70},
	"PSB-SWB": {DistanceKm: 27.6, AvgDurationMin: 70},

	"SWB-PST": {DistanceKm: 32.6, AvgDurationMin: 70},
	"PST-SWB": {DistanceKm: 32.6, AvgDurationMin: 70},
}
