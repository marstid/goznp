---
name: Bug report
about: Create a report to help us improve
title: '[BUG] '
labels: bug
---

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Run command '...'
2. Provide device information '...'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Actual behavior**
What actually happened. Include error messages if any.

**Device Information**
- Device model and manufacturer:
- Device type (e.g., bulb, sensor, switch):
- Zigbee cluster(s) involved:

**Environment**
- Go version: `go version`
- goznp version:
- Operating system (e.g., Linux, macOS, Windows):
- Coordinator model (e.g., CC2652R, CC1352P):
- Firmware (e.g., Koenkk Z-Stack 3.x.0):

**Command used**
```bash
# Paste the exact command you ran
go run main.go ...
```

**Log output**
```bash
# Paste relevant log output here
```

**Debug output (if applicable)**
Run with `GOZNP_DEBUG=1` environment variable and paste the debug output:
```bash
GOZNP_DEBUG=1 go run main.go ...
```

**Device info**
If applicable, run `goznp device info --addr 0xXXXX --verbose` and paste the output:

**Additional context**
Add any other context about the problem here.

- Did this work before? If so, what changed?
- Are you using any custom device quirks?
- Any relevant network details (e.g., many devices, channel congestion)?
