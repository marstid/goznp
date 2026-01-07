package devices

// Grouped by raw manufacturer ID as reported by devices.
var deviceDatabase = map[Fingerprint]Info{
	//nolint:godot // Section divider.
	// ==========================================================================
	// _TZ3000_* (Tuya Zigbee 3.0 devices).
	// ==========================================================================
	{"_TZ3000_266azbg3", "TS011F"}: {"NOUS", "A6Z", "Outdoor smart socket"},
	{"_TZ3000_w0qqde0g", "TS011F"}: {"Tuya", "TS011F", "Smart plug"},
	{"_TZ3000_amdymr7l", "TS011F"}: {"BlitzWolf", "BW-SHP13", "Smart plug with power monitoring"},
	{"_TZ3000_typdpbpg", "TS011F"}: {"Tuya", "TS011F", "Smart plug with power monitoring"},
	{"_TZ3000_cphmq0q7", "TS011F"}: {"Moes", "TS011F", "Smart plug with power monitoring"},
	{"_TZ3000_ew3ldmgx", "TS011F"}: {"Tuya", "TS011F", "Smart plug"},
	{"_TZ3000_gjnozsaz", "TS011F"}: {"Tuya", "TS011F", "Smart plug with power monitoring"},
	{"_TZ3000_dpo1ysak", "TS011F"}: {"Tuya", "TS011F", "Smart plug"},
	{"_TZ3000_okaz9tjs", "TS011F"}: {"Silvercrest", "HG06337", "Smart plug"},
	{"_TZ3000_1h2x4akh", "TS011F"}: {"Tuya", "TS011F", "Smart plug with power monitoring"},
	{"_TZ3000_b28wrpvx", "TS011F"}: {"Tuya", "TS011F", "Smart plug"},
	{"_TZ3000_cehuw1lw", "TS011F"}: {"Tuya", "TS011F", "Smart plug with power monitoring"},
	{"_TZ3000_5f43h46b", "TS011F"}: {"Tuya", "TS011F", "Smart plug with power monitoring"},
	{"_TZ3000_rdtixbnu", "TS011F"}: {"Tuya", "TS011F", "Smart plug power strip"},
	{"_TZ3000_cfnprab5", "TS011F"}: {"Nous", "A1Z", "Smart plug"},
	{"_TZ3000_ksyx5swy", "TS011F"}: {"Tuya", "TS011F", "Smart power strip 4 AC + 2 USB"},
	{"_TZ3000_zloso4jk", "TS011F"}: {"Nous", "A4Z", "Smart plug with USB"},
	{"_TZ3000_qaaysllp", "TS0201"}: {"Tuya", "TS0201", "Temperature & humidity sensor"},
	{"_TZ3000_bguser20", "TS0201"}: {"Tuya", "TS0201", "Temperature & humidity sensor"},
	{"_TZ3000_xr3htd96", "TS0201"}: {"Tuya", "TS0201", "Temperature & humidity sensor"},
	{"_TZ3000_fllyghyj", "TS0201"}: {"Tuya", "TS0201", "Temperature & humidity sensor"},
	{"_TZ3000_dowj6gyi", "TS0203"}: {"Tuya", "TS0203", "Door/window sensor"},
	{"_TZ3000_26fmupbb", "TS0203"}: {"Tuya", "TS0203", "Door/window sensor"},
	{"_TZ3000_402jjyro", "TS0203"}: {"Tuya", "TS0203", "Door/window sensor"},
	{"_TZ3000_mcxw5ehu", "TS0202"}: {"Tuya", "TS0202", "Motion sensor"},
	{"_TZ3000_6ygjfyll", "TS0202"}: {"Tuya", "TS0202", "Motion sensor"},

	//nolint:godot // Section divider.
	// ==========================================================================
	// eWeLink (Sonoff).
	// ==========================================================================
	{"eWeLink", "SA-003-Zigbee"}: {"Sonoff", "SA-003", "Smart switch"},
	{"eWeLink", "TH01"}:          {"Sonoff", "SNZB-02", "Temperature & humidity sensor"},

	//nolint:godot // Section divider.
	// ==========================================================================
	// GLEDOPTO.
	// ==========================================================================
	{"GLEDOPTO", "GL-SD-301P"}:  {"GLEDOPTO", "GL-SD-301P", "Zigbee AC dimmer"},
	{"GLEDOPTO", "GL-C-006"}:    {"GLEDOPTO", "GL-C-006", "Zigbee LED controller RGB+CCT"},
	{"GLEDOPTO", "GL-C-007"}:    {"GLEDOPTO", "GL-C-007", "Zigbee LED controller RGBW"},
	{"GLEDOPTO", "GL-C-008"}:    {"GLEDOPTO", "GL-C-008", "Zigbee LED controller RGB+CCT"},
	{"GLEDOPTO", "GL-B-001Z"}:   {"GLEDOPTO", "GL-B-001Z", "Zigbee smart bulb E27 RGB+CCT"},
	{"GLEDOPTO", "GL-B-007Z"}:   {"GLEDOPTO", "GL-B-007Z", "Zigbee smart bulb E27 RGBW"},
	{"GLEDOPTO", "GL-B-008Z"}:   {"GLEDOPTO", "GL-B-008Z", "Zigbee smart bulb E27 RGB+CCT"},
	{"GLEDOPTO", "GL-S-007Z"}:   {"GLEDOPTO", "GL-S-007Z", "Zigbee smart spot GU10 RGBW"},
	{"GLEDOPTO", "GL-D-003Z"}:   {"GLEDOPTO", "GL-D-003Z", "Zigbee LED downlight RGB+CCT"},
	{"GLEDOPTO", "GL-FL-004TZ"}: {"GLEDOPTO", "GL-FL-004TZ", "Zigbee outdoor floodlight"},

	// ==========================================================================.
	// IKEA of Sweden.
	// ==========================================================================.
	{"IKEA of Sweden", "TRADFRI bulb E27 WS opal 980lm"}:  {"IKEA", "LED1545G12", "TRADFRI bulb E27 980lm"},
	{"IKEA of Sweden", "TRADFRI bulb E27 WS opal 1000lm"}: {"IKEA", "LED1732G11", "TRADFRI bulb E27 1000lm"},
	{"IKEA of Sweden", "TRADFRI bulb E27 CWS opal 600lm"}: {"IKEA", "LED1624G9", "TRADFRI bulb E27 color 600lm"},
	{"IKEA of Sweden", "TRADFRI bulb E27 CWS 806lm"}:      {"IKEA", "LED1924G9", "TRADFRI bulb E27 color 806lm"},
	{"IKEA of Sweden", "TRADFRI bulb E14 WS opal 400lm"}:  {"IKEA", "LED1536G5", "TRADFRI bulb E14 400lm"},
	{"IKEA of Sweden", "TRADFRI bulb E14 WS 470lm"}:       {"IKEA", "LED1835C6", "TRADFRI bulb E14 470lm"},
	{"IKEA of Sweden", "TRADFRI bulb GU10 WS 400lm"}:      {"IKEA", "LED1837R5", "TRADFRI bulb GU10 400lm"},
	{"IKEA of Sweden", "TRADFRI bulb E27 W opal 1000lm"}:  {"IKEA", "LED1623G12", "TRADFRI bulb E27 warm 1000lm"},
	{"IKEA of Sweden", "TRADFRI control outlet"}:          {"IKEA", "E1603", "TRADFRI control outlet"},
	{"IKEA of Sweden", "TRADFRI remote control"}:          {"IKEA", "E1524", "TRADFRI remote control"},
	{"IKEA of Sweden", "TRADFRI on/off switch"}:           {"IKEA", "E1743", "TRADFRI on/off switch"},
	{"IKEA of Sweden", "TRADFRI motion sensor"}:           {"IKEA", "E1525", "TRADFRI motion sensor"},
	{"IKEA of Sweden", "TRADFRI signal repeater"}:         {"IKEA", "E1746", "TRADFRI signal repeater"},
	{"IKEA of Sweden", "TRADFRI wireless dimmer"}:         {"IKEA", "ICTC-G-1", "TRADFRI wireless dimmer"},
	{"IKEA of Sweden", "FYRTUR block-out roller blind"}:   {"IKEA", "E1757", "FYRTUR roller blind"},
	{"IKEA of Sweden", "KADRILJ roller blind"}:            {"IKEA", "E1926", "KADRILJ roller blind"},
	{"IKEA of Sweden", "SYMFONISK sound controller"}:      {"IKEA", "E1744", "SYMFONISK sound controller"},
	{"IKEA of Sweden", "RODRET dimmer"}:                   {"IKEA", "E2201", "RODRET dimmer"},
	{"IKEA of Sweden", "STYRBAR remote control N2"}:       {"IKEA", "E2002", "STYRBAR remote control"},
	{"IKEA of Sweden", "PARASOLL door/window sensor"}:     {"IKEA", "E2013", "PARASOLL door/window sensor"},
	{"IKEA of Sweden", "VALLHORN motion sensor"}:          {"IKEA", "E2134", "VALLHORN motion sensor"},

	// ==========================================================================.
	// LUMI (Aqara / Xiaomi).
	// ==========================================================================.
	{"LUMI", "lumi.sensor_ht"}:         {"Aqara", "WSDCGQ01LM", "Temperature & humidity sensor"},
	{"LUMI", "lumi.weather"}:           {"Aqara", "WSDCGQ11LM", "Temperature & humidity sensor"},
	{"LUMI", "lumi.sensor_ht.agl02"}:   {"Aqara", "WSDCGQ12LM", "Temperature & humidity sensor"},
	{"LUMI", "lumi.sensor_motion"}:     {"Aqara", "RTCGQ01LM", "Motion sensor"},
	{"LUMI", "lumi.sensor_motion.aq2"}: {"Aqara", "RTCGQ11LM", "Motion sensor"},
	{"LUMI", "lumi.sensor_magnet"}:     {"Aqara", "MCCGQ01LM", "Door/window sensor"},
	{"LUMI", "lumi.sensor_magnet.aq2"}: {"Aqara", "MCCGQ11LM", "Door/window sensor"},
	{"LUMI", "lumi.sensor_switch"}:     {"Aqara", "WXKG01LM", "Wireless switch"},
	{"LUMI", "lumi.sensor_switch.aq2"}: {"Aqara", "WXKG11LM", "Wireless switch"},
	{"LUMI", "lumi.sensor_cube"}:       {"Aqara", "MFKZQ01LM", "Cube controller"},
	{"LUMI", "lumi.plug"}:              {"Aqara", "ZNCZ02LM", "Smart plug"},
	{"LUMI", "lumi.plug.maeu01"}:       {"Aqara", "SP-EUC01", "Smart plug EU"},
	{"LUMI", "lumi.ctrl_ln1.aq1"}:      {"Aqara", "QBKG11LM", "Wall switch (no neutral)"},
	{"LUMI", "lumi.ctrl_ln2.aq1"}:      {"Aqara", "QBKG12LM", "Wall switch double (no neutral)"},
	{"LUMI", "lumi.ctrl_neutral1"}:     {"Aqara", "QBKG04LM", "Wall switch single"},
	{"LUMI", "lumi.ctrl_neutral2"}:     {"Aqara", "QBKG03LM", "Wall switch double"},
	{"LUMI", "lumi.vibration.aq1"}:     {"Aqara", "DJT11LM", "Vibration sensor"},
	{"LUMI", "lumi.sensor_water.aq1"}:  {"Aqara", "SJCGQ11LM", "Water leak sensor"},

	// ==========================================================================.
	// Philips.
	// ==========================================================================.
	{"Philips", "LWB010"}: {"Philips Hue", "LWB010", "White bulb A19"},
	{"Philips", "LWB014"}: {"Philips Hue", "LWB014", "White bulb A19"},
	{"Philips", "LWA001"}: {"Philips Hue", "LWA001", "White bulb A19 800lm"},
	{"Philips", "LWA004"}: {"Philips Hue", "LWA004", "White bulb A19 800lm"},
	{"Philips", "LWA019"}: {"Philips Hue", "LWA019", "White bulb A67 1600lm"},
	{"Philips", "LCT001"}: {"Philips Hue", "LCT001", "Color bulb A19"},
	{"Philips", "LCT007"}: {"Philips Hue", "LCT007", "Color bulb A19"},
	{"Philips", "LCT010"}: {"Philips Hue", "LCT010", "Color bulb A19"},
	{"Philips", "LCT015"}: {"Philips Hue", "LCT015", "Color bulb A19"},
	{"Philips", "LCA001"}: {"Philips Hue", "LCA001", "Color bulb A19"},
	{"Philips", "LCA003"}: {"Philips Hue", "LCA003", "Color bulb A19"},
	{"Philips", "LCG002"}: {"Philips Hue", "LCG002", "Color spot GU10"},
	{"Philips", "LWG004"}: {"Philips Hue", "LWG004", "White spot GU10"},
	{"Philips", "LST002"}: {"Philips Hue", "LST002", "Lightstrip Plus"},
	{"Philips", "LCL001"}: {"Philips Hue", "LCL001", "Lightstrip Plus"},
	{"Philips", "SML001"}: {"Philips Hue", "SML001", "Motion sensor"},
	{"Philips", "SML002"}: {"Philips Hue", "SML002", "Motion sensor outdoor"},
	{"Philips", "RWL021"}: {"Philips Hue", "RWL021", "Dimmer switch"},
	{"Philips", "RWL022"}: {"Philips Hue", "RWL022", "Dimmer switch v2"},

	// ==========================================================================.
	// Signify Netherlands B.V. (Philips Hue)
	// ==========================================================================.
	{"Signify Netherlands B.V.", ""}: {"Philips Hue", "", "Hue device"},

	// ==========================================================================.
	// SONOFF
	// ==========================================================================.
	{"SONOFF", "BASICZBR3"}:   {"Sonoff", "BASICZBR3", "Smart switch"},
	{"SONOFF", "S31 Lite zb"}: {"Sonoff", "S31ZB", "Smart plug"},
	{"SONOFF", "S26R2ZB"}:     {"Sonoff", "S26R2ZB", "Smart plug"},
	{"SONOFF", "SNZB-01"}:     {"Sonoff", "SNZB-01", "Wireless button"},
	{"SONOFF", "SNZB-02"}:     {"Sonoff", "SNZB-02", "Temperature & humidity sensor"},
	{"SONOFF", "SNZB-03"}:     {"Sonoff", "SNZB-03", "Motion sensor"},
	{"SONOFF", "SNZB-04"}:     {"Sonoff", "SNZB-04", "Door/window sensor"},
	{"SONOFF", "SNZB-01P"}:    {"Sonoff", "SNZB-01P", "Wireless button"},
	{"SONOFF", "SNZB-02P"}:    {"Sonoff", "SNZB-02P", "Temperature & humidity sensor"},
	{"SONOFF", "SNZB-03P"}:    {"Sonoff", "SNZB-03P", "Motion sensor"},
	{"SONOFF", "SNZB-04P"}:    {"Sonoff", "SNZB-04P", "Door/window sensor"},
	{"SONOFF", "SNZB-06P"}:    {"Sonoff", "SNZB-06P", "Occupancy sensor"},
	{"SONOFF", "ZBMINI"}:      {"Sonoff", "ZBMINI", "Smart switch"},
	{"SONOFF", "ZBMINIL2"}:    {"Sonoff", "ZBMINIL2", "Smart switch (no neutral)"},
}
