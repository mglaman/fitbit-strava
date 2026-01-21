package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"fitbit-strava/auth"
	"fitbit-strava/config"
	"fitbit-strava/encoder"
	"fitbit-strava/fitbit"
	"fitbit-strava/strava"

	"github.com/charmbracelet/huh"
	"github.com/dustin/go-humanize"
	"golang.org/x/oauth2"
	fitbitOAuth "golang.org/x/oauth2/fitbit"
)

func main() {
	// Flags
	startTimeStr := flag.String("start", "", "Start time (HH:mm)")
	durationMin := flag.Int("duration", 0, "Duration in minutes")
	// Date defaults to today if not provided via flag, but we'll check if it was explicitly set later if needed,
	// or just use the current value as default for the prompt.
	dateStr := flag.String("date", "", "Date (YYYY-MM-DD)")
	dryRun := flag.Bool("dry-run", false, "Generate FIT file but do not upload to Strava")
	flag.Parse()

	// 1. Load Config & Auth EARLY
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	tokenStore, err := auth.LoadTokens()
	if err != nil {
		log.Fatalf("Error loading token store: %v", err)
	}
	authenticator := auth.NewAuthenticator(tokenStore)

	// 2. Authenticate Services
	// Fitbit
	fitbitConfig := &oauth2.Config{
		ClientID:     cfg.FitbitClientID,
		ClientSecret: cfg.FitbitClientSecret, // Fixed typo in variable name if strictly following config
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"heartrate", "activity"},
		Endpoint:     fitbitOAuth.Endpoint,
	}
	fitbitClient := fitbit.NewClient(authenticator.GetClient(context.Background(), "fitbit", fitbitConfig))

	// Strava
	stravaEndpoint := oauth2.Endpoint{
		AuthURL:  "https://www.strava.com/oauth/mobile/authorize",
		TokenURL: "https://www.strava.com/oauth/token",
	}
	stravaOAuth := &oauth2.Config{
		ClientID:     cfg.StravaClientID,
		ClientSecret: cfg.StravaClientSecret,
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"activity:write"},
		Endpoint:     stravaEndpoint,
	}
	stravaClient := strava.NewClient(authenticator.GetClient(context.Background(), "strava", stravaOAuth))

	// Interactive Mode
	interactive := false
	var selectedActivity *fitbit.ActivityLog

	if *startTimeStr == "" {
		interactive = true

		// 1. Activity Selection
		// 0 = Manual, >0 = Activity Index (but we will use the pointer directly if possible or an ID)
		// 0 = Manual, >0 = Activity Index (but we will use the pointer directly if possible or an ID)
		// Let's use string keys for selection: "manual" or "logId"
		var selectedAction string

		// Fetch recent activities
		recent, err := fitbitClient.GetRecentActivities(15)
		if err == nil && len(recent.Activities) > 0 {
			filteredActivities := make([]fitbit.ActivityLog, 0, len(recent.Activities))
			for _, act := range recent.Activities {
				// Exclude activities with GPS as they typically sync automatically to Strava
				if act.HasGPS {
					continue
				}
				filteredActivities = append(filteredActivities, act)
			}

			if len(filteredActivities) > 0 {
				options := make([]huh.Option[string], 0, len(filteredActivities)+1)
				for _, act := range filteredActivities {
					// Parse time for nicer display
					t, _ := time.Parse("2006-01-02T15:04:05.000-07:00", act.StartTime)
					relativeTime := humanize.Time(t)
					displayTime := t.Format("15:04") // Just time, date is in relative

					durMin := act.Duration / 60000

					// e.g., "Strength Training - 2 hours ago (13:00) [304 cal]"
					label := fmt.Sprintf("%s - %s (%s) [%dm, %d cal]", act.Name, relativeTime, displayTime, durMin, act.Calories)
					// Use LogID as value, converted to string
					options = append(options, huh.NewOption(label, strconv.FormatInt(act.LogID, 10)))
				}
				// Add manual entry option
				options = append(options, huh.NewOption("Manual Entry", "manual"))

				form := huh.NewForm(
					huh.NewGroup(
						huh.NewSelect[string]().
							Title("Select Fitbit Activity").
							Options(options...).
							Value(&selectedAction),
					),
				)
				err = form.Run()
				if err != nil {
					log.Fatal("Selection cancelled")
				}

				if selectedAction != "manual" {
					// Find the selected activity object
					id, _ := strconv.ParseInt(selectedAction, 10, 64)
					for _, act := range filteredActivities {
						if act.LogID == id {
							selectedActivity = &act
							break
						}
					}

					if selectedActivity != nil {
						t, err := time.Parse("2006-01-02T15:04:05.000-07:00", selectedActivity.StartTime)
						if err != nil {
							log.Printf("Error parsing time: %v", err)
							// Fallback to manual if parsing fails? Or just continue with manual
							selectedAction = "manual"
						} else {
							*dateStr = t.Format("2006-01-02")
							*startTimeStr = t.Format("15:04")
							*durationMin = selectedActivity.Duration / 60000
						}
					}
				}
			} else {
				fmt.Println("No suitable recent activities found.")
				selectedAction = "manual"
			}
		} else {
			if err != nil {
				fmt.Printf("Error fetching recent activities: %v\n", err)
			}
			selectedAction = "manual"
		}

		if selectedAction == "manual" {
			// Manual Entry Form
			if *dateStr == "" {
				*dateStr = time.Now().Format("2006-01-02")
			}
			if *durationMin == 0 {
				*durationMin = 60
			}
			durationStr := strconv.Itoa(*durationMin)

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Date").
						Description("YYYY-MM-DD").
						Value(dateStr).
						Validate(func(str string) error {
							_, err := time.Parse("2006-01-02", str)
							return err
						}),
					huh.NewInput().
						Title("Start Time").
						Description("HH:mm").
						Value(startTimeStr).
						Validate(func(str string) error {
							_, err := time.Parse("15:04", str)
							return err
						}),
					huh.NewInput().
						Title("Duration (minutes)").
						Value(&durationStr).
						Validate(func(str string) error {
							_, err := strconv.Atoi(str)
							return err
						}),
				),
			)

			err := form.Run()
			if err != nil {
				log.Fatal("Input cancelled")
			}
			*durationMin, _ = strconv.Atoi(durationStr)
		}
	} else {
		// If non-interactive, set default date if missing
		if *dateStr == "" {
			*dateStr = time.Now().Format("2006-01-02")
		}
		// Set default duration if missing
		if *durationMin == 0 {
			*durationMin = 60
		}
	}

	// 3. Metadata Calculation
	start, err := time.Parse("15:04", *startTimeStr)
	if err != nil {
		log.Fatalf("Invalid start time format: %v", err)
	}
	end := start.Add(time.Duration(*durationMin) * time.Minute)
	endTimeStr := end.Format("15:04")

	fmt.Printf("Fetching heart rate data for %s from %s to %s...\n", *dateStr, *startTimeStr, endTimeStr)

	// 4. Fetch Data
	hrData, err := fitbitClient.FetchIntradayHeartRate(*dateStr, *startTimeStr, endTimeStr)
	if err != nil {
		log.Fatalf("Failed to fetch Fitbit data: %v", err)
	}
	fmt.Printf("Retrieved %d heart rate samples.\n", len(hrData.ActivitiesHeartIntraday.Dataset))

	if len(hrData.ActivitiesHeartIntraday.Dataset) == 0 {
		log.Println("No data found for this period. Exiting.")
		return
	}

	// 4b. Fetch Activity Logs for Calories and Source
	var totalCalories int
	var activitySource *fitbit.ActivityLogSource
	var activityName string = "Workout"

	// If we selected an activity interactively, use its name as efficient default
	if interactive && selectedActivity != nil {
		activityName = selectedActivity.Name
	}

	activityLogs, err := fitbitClient.GetActivityLogs(*dateStr)
	if err != nil {
		log.Printf("Warning: Failed to fetch activity logs: %v\n", err)
	} else {
		// Find matching log
		// Fitbit logs use "13:00" format. *startTimeStr is "HH:mm".
		for _, logItem := range activityLogs.Activities {
			// Simple match on start time. Could allow fuzzy matching later.
			if logItem.StartTime == *startTimeStr {
				totalCalories = logItem.Calories
				// Use the source from the log if available
				// (in interactive mode we could have passed it, but re-fetching here is consistent)
				activitySource = &logItem.Source
				activityName = logItem.Name
				fmt.Printf("Found matching activity log: %s (Calories: %d)\n", logItem.Name, totalCalories)
				break
			}
		}
	}

	// 5. Create FIT File
	fitFilename := "workout.fit"
	fmt.Println("Generating FIT file...")
	if err := encoder.CreateFitFile(fitFilename, *dateStr, *startTimeStr, "", hrData, totalCalories, activitySource, activityName); err != nil {
		log.Fatalf("Failed to create FIT file: %v", err)
	}
	fmt.Println("FIT file created successfully.")

	// 6. Upload to Strava
	if *dryRun {
		fmt.Println("Dry run enabled. Skipping Strava upload.")
		fmt.Printf("File saved to %s\n", fitFilename)
		return
	}

	// Interactive Confirmation
	if interactive {
		var confirm bool
		err := huh.NewConfirm().
			Title("Ready to upload to Strava?").
			Value(&confirm).
			Run()

		if err != nil {
			log.Fatal("Confirmation cancelled")
		}
		if !confirm {
			fmt.Println("Upload cancelled.")
			return
		}
	}

	fmt.Println("Uploading to Strava...")

	// Determine default name based on time of day
	hour := start.Hour()
	var defaultName string
	switch {
	case hour >= 4 && hour < 12:
		defaultName = "Morning workout ☀️"
	case hour >= 12 && hour < 17:
		defaultName = "Afternoon workout 💪"
	case hour >= 17 && hour < 21:
		defaultName = "Evening workout 🌙"
	default:
		defaultName = "Night workout 🌚"
	}

	// Create metadata
	metadata := strava.ActivityMetadata{
		Name: defaultName,
	}
	if totalCalories > 0 {
		// metadata.Description = fmt.Sprintf("Imported from Fitbit. Total Calories: %d", totalCalories)
		// If we found a named activity log, use its name
		for _, logItem := range activityLogs.Activities {
			if logItem.StartTime == *startTimeStr {
				// If the log name is generic "Workout", keep our time-based name.
				// Otherwise, use the specific name.
				if logItem.Name != "Workout" && logItem.Name != "Activity" {
					metadata.Name = logItem.Name
				}
				metadata.ExternalID = fmt.Sprintf("fitbit-%d", logItem.LogID)
				break
			}
		}
	}

	resp, err := stravaClient.UploadActivity(fitFilename, metadata)
	if err != nil {
		log.Fatalf("Failed to upload to Strava: %v", err)
	}
	fmt.Printf("Upload successful! Response: %s\n", resp)

	// Cleanup
	// os.Remove(fitFilename)
}
