---
name: Device support request
about: Request support for a new Zigbee device
title: '[DEVICE] '
labels: device
---

**Device Information**
- **Model:** (e.g., TUOS-1609)
- **Manufacturer:** (e.g., Tuya)
- **Device Name:** (e.g., 4-Button Remote)
- **Device Type:** (e.g., Bulb, Sensor, Switch, Remote)
- **Device ID:** (e.g., 0x0201 for thermostat, 0x0100 for on/off light)

**Description**
A brief description of the device and its capabilities (number of buttons, power source, etc.).

**Device Fingerprint**
Please run `goznp device info --addr 0xXXXX --verbose` and paste the full output:
```bash
# Paste goznp device info output here
```

This should include:
- IEEE address
- Network address
- Manufacturer name
- Model name
- List of endpoints and their clusters

**Device Behavior**
Describe how the device currently behaves with goznp:

- Does it pair successfully?
- Can you control it?
- Do you receive updates/notifications from it?
- What works and what doesn't?

**Expected Behavior**
What you expected to happen with this device.

**Zigbee2MQTT / Other Integrations**
If this device works with other Zigbee libraries, please provide links to their configuration:

- Zigbee2MQTT device page URL:
- zigbee-herdsman device support:
- Other integration:

**Logs**
Please run with `GOZNP_DEBUG=1` and capture logs showing:
1. Device pairing process
2. Attempted communication with the device
3. Any error messages

```bash
# Paste relevant debug logs
```

**Files / Attachments**
- Screenshot of device interview output (if applicable)
- Copy of device-specific quirks or configuration (if you found one)

**Additional Information**
Any other details that might help add support for this device:

- The device doesn't respond to read commands
- Device doesn't support specific cluster
- Device requires custom handling (e.g., specific command format)
- Device quirks you've discovered

**Willing to test?**
- [ ] Yes, I can help test any changes to add support
- [ ] I can provide more logs if needed
