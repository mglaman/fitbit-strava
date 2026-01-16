# Agent Context: Fitbit to Strava Bridge

This document provides context and architectural decisions for AI agents working on this codebase.

## Project Goal
A Go CLI tool to fetch heart rate data from Fitbit and upload it to Strava as a "Weight Training" activity, bridging the gap for non-GPS activities.

## Architecture
- **`main.go`**: Orchestrates the flow. Handles CLI flags, interactive prompts, and high-level logic (e.g., activity naming heuristics).
- **`auth/`**: Handles OAuth2 for both providers. Uses `credentials.json` for persistence.
- **`fitbit/`**: Client for fetching Intraday Heart Rate (1sec) and Activity Logs.
- **`encoder/`**: Generates FIT files using `github.com/tormoder/fit`.
- **`strava/`**: Client for multipart file uploads.

## Key Design Decisions

### 1. Activity Type Hardcoding
We explicitly set the FIT file metadata to "Weight Training" because that is the user's primary use case.
- **Sport**: `fit.SportTraining` (10)
- **SubSport**: `fit.SubSportStrengthTraining` (20)
- **SportProfileName**: "Weight Training" (Visible string)

### 2. Time-Based Naming
In `main.go`, we implement a heuristic to name activities (e.g., "Morning workout ☀️") based on the hour of the day. This is overridden *only* if a specific, non-generic name is found in the Fitbit Activity CLI.

### 3. FIT File timestamps
FIT files require `uint32` timestamps scaled to milliseconds for `TotalTimerTime` and `TotalElapsedTime`. We calculate this based on the first and last heart rate sample to ensure validity.

### 4. Interactive Mode
The tool detects if flags are missing and switches to `bufio` readers for interactive input.

## Common Tasks
- **Adding new metadata**: Update `strava.ActivityMetadata` struct and `main.go` logic.
- **Changing activity type**: Modify `encoder/encoder.go`.
