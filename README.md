# Fitbit to Strava Bridge

A CLI tool written in Go that fetches high-resolution heart rate data from Fitbit and uploads it to Strava.

**Perfect for:** Strength training, spinning, treadmill runs, yoga, and any other indoor activity where you might not record GPS data but want your heart rate and effort to be accurately reflected in Strava.

## Features

- **Activity Sync**: Automatically fetches your recent non-GPS activities from Fitbit.
- **High-Resolution Data**: Retrieves 1-second interval heart rate data for precise analysis.
- **Smart Metadata**:
    - **Device Info**: Correctly identifies your device (e.g., "Fitbit Charge 6") in the upload.
    - **Dynamic Sports**: Maps Fitbit activities (Spinning, Yoga, Weights) to the correct Strava sport types.
    - **Rich Naming**: content-aware naming conventions.
- **Interactive CLI**: A modern terminal interface for selecting activities to sync.
- **Dry Run**: Validate the generated FIT file before uploading.

## Installation

1.  Clone the repository.
2.  Build the binary:
    ```bash
    go build -o fitbit-strava .
    ```

## Prerequisites

You need to create your own "Applications" on Fitbit and Strava to get the necessary API keys.

### 1. Fitbit App
1.  Go to [dev.fitbit.com](https://dev.fitbit.com/apps/new) and register a new app.
2.  **OAuth 2.0 Application Type**: Select **Personal** (this is crucial for accessing Intraday heart rate data).
3.  **Callback URL**: `http://localhost:8080/callback`
4.  Copy your **Client ID** and **Client Secret**.

### 2. Strava App
1.  Go to [Strava API Settings](https://www.strava.com/settings/api) and create an application.
2.  **Authorization Callback Domain**: `localhost`
3.  Copy your **Client ID** and **Client Secret**.

## Configuration

Create a `.env` file in the project root with your OAuth2 credentials:

```env
FITBIT_CLIENT_ID=your_fitbit_client_id
FITBIT_CLIENT_SECRET=your_fitbit_client_secret

STRAVA_CLIENT_ID=your_strava_client_id
STRAVA_CLIENT_SECRET=your_strava_client_secret
```

## Usage

### Interactive Mode
Run the tool to see a list of your recent eligible activities:

```bash
./fitbit-strava
```

Use the arrow keys to select an activity and Enter to confirm.

### Manual Mode
You can also manually specify a time range if you haven't logged it in Fitbit yet:

```bash
./fitbit-strava -start 18:30 -duration 45
```

### Options
- `-start`: Start time (HH:mm)
- `-duration`: Duration in minutes (default: 60)
- `-date`: Date (YYYY-MM-DD, default: today)
- `-dry-run`: Generate `workout.fit` but skip upload.

## Authentication
On first run, the tool will open your browser to authenticate with both Fitbit and Strava. Tokens are saved locally to `credentials.json`.
