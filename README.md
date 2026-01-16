# Fitbit to Strava Bridge for Weight Training

A CLI tool written in Go that fetches high-resolution heart rate data from Fitbit and uploads it to Strava as a "Weight Training" activity.

This bridges the gap for strength training activities where GPS is not needed (and often not recorded) but heart rate data is valuable for Strava's "Relative Effort" and fitness tracking.

## Features

- **High-Resolution Data**: Fetches 1-second interval heart rate data from Fitbit.
- **Automated Metadata**: automatically sets activity type to "Weight Training".
- **Dynamic Naming**: Names your workout based on the time of day (e.g., "Morning workout ☀️") or uses the specific activity name if logged in Fitbit.
- **Calorie Sync**: Syncs "Total Calories" if a matching activity log exists in Fitbit.
- **Interactive Mode**: Prompts for details if flags are not provided.
- **Dry Run**: Validate the generated FIT file before uploading.

## Installation

1.  Clone the repository.
2.  Build the binary:
    ```bash
    go build -o fitbit-strava .
    ```

## Configuration

Create a `.env` file in the project root with your OAuth2 credentials:

```env
FITBIT_CLIENT_ID=your_fitbit_client_id
FITBIT_CLIENT_SECRET=your_fitbit_client_secret
FITBIT_REDIRECT_URL=http://localhost:8080/callback

STRAVA_CLIENT_ID=your_strava_client_id
STRAVA_CLIENT_SECRET=your_strava_client_secret
STRAVA_REDIRECT_URL=http://localhost:8080/callback
```

## Usage

### Interactive Mode
Simply run the tool and follow the prompts:

```bash
./fitbit-strava
```

### CLI Flags
Quickly run with specific parameters:

```bash
./fitbit-strava -start 18:30 -duration 45
```

**Options:**
- `-start`: Start time (HH:mm)
- `-duration`: Duration in minutes (default: 60)
- `-date`: Date (YYYY-MM-DD, default: today)
- `-dry-run`: Generate `workout.fit` but skip upload.

## Authentication
On first run, the tool will open your browser to authenticate with both Fitbit and Strava. Tokens are saved locally to `credentials.json`.
