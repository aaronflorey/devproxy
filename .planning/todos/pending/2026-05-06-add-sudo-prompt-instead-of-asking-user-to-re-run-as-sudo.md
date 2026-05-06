---
created: 2026-05-06T03:56:48.272Z
title: Add sudo prompt instead of asking user to re-run as sudo
area: general
files: []
---

## Problem

The current flow asks the user to manually re-run commands with sudo when elevated privileges are required. This creates friction, interrupts workflow, and can lead to repeated failed attempts before the user realizes elevation is needed.

## Solution

Update privileged operations to trigger a sudo prompt automatically at execution time (for example via a preflight privilege check and elevation request) rather than failing and instructing the user to re-run manually. Ensure errors remain explicit when elevation is denied or unavailable.
