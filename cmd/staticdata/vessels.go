package staticdata

/*
 * Vessel MMSI mapping and vessel name lookup functions
 *
 * Data source: Manually curated from BC Ferries fleet information
 * MMSI = Maritime Mobile Service Identity (unique vessel identifier)
 */

// vesselMMSI maps vessel names to their MMSI numbers
var vesselMMSI = map[string]int{
	// Tsawwassen to Southern Gulf Islands
	"Salish Raven":                316030628,
	"Salish Heron":                316047943,
	"Salish Eagle":                316030626,

	// Vancouver (Tsawwassen) to Victoria (Swartz Bay)
	"Spirit of British Columbia":  316001268,
	"Spirit of Vancouver Island":  316001269,

	// Vancouver (Tsawwassen) to Fulford Harbour (Salt Spring Island)
	"Skeena Queen":                316001267,

	// Victoria (Swartz Bay) to Southern Gulf Islands
	"Queen of Cumberland":         316001252,

	// Add more vessels as needed
}

// mmsiToVessel is a reverse lookup map (MMSI → vessel name)
// Built automatically in init()
var mmsiToVessel map[int]string

func init() {
	// Build reverse lookup map for MMSI → vessel name
	mmsiToVessel = make(map[int]string)
	for name, mmsi := range vesselMMSI {
		mmsiToVessel[mmsi] = name
	}
}

/*
 * GetVesselMMSI
 *
 * Returns the MMSI for a vessel name, or nil if not found.
 * Vessel name matching is case-sensitive.
 *
 * @param vesselName string - The vessel name (e.g., "Salish Eagle")
 *
 * @return *int - Pointer to MMSI number, or nil if not found
 */
func GetVesselMMSI(vesselName string) *int {
	if mmsi, ok := vesselMMSI[vesselName]; ok {
		return &mmsi
	}
	return nil
}

/*
 * GetVesselNameByMMSI
 *
 * Returns the vessel name for a given MMSI, or empty string if not found.
 *
 * @param mmsi int - The MMSI number (e.g., 316030626)
 *
 * @return string - Vessel name, or empty string if not found
 */
func GetVesselNameByMMSI(mmsi int) string {
	return mmsiToVessel[mmsi]
}
