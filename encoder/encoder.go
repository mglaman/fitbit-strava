package encoder

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"time"

	"fitbit-strava/fitbit"

	"github.com/tormoder/fit"
)

func CreateFitFile(filename string, date, startTime, durationStr string, data *fitbit.HeartRateResponse, totalCalories int, source *fitbit.ActivityLogSource, activityName string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	// Parse base time
	baseTime, err := time.Parse("2006-01-02 15:04", date+" "+startTime)
	if err != nil {
		return fmt.Errorf("failed to parse start time: %v", err)
	}

	// Create FIT activity file
	// Create Header
	hdr := fit.NewHeader(fit.V20, true)

	fitFile, err := fit.NewFile(fit.FileTypeActivity, hdr)
	if err != nil {
		return fmt.Errorf("failed to create fit file: %v", err)
	}

	// Add FileCreator message (Development Manufacturer)
	fileCreator := fit.NewFileCreatorMsg()
	fileCreator.SoftwareVersion = 1
	fitFile.FileCreator = fileCreator

	// Set FileId TimeCreated to Now (Export time)
	fitFile.FileId.TimeCreated = time.Now()
	fitFile.FileId.Manufacturer = fit.ManufacturerDevelopment
	fitFile.FileId.Product = 0 // Generic

	activity, err := fitFile.Activity()
	if err != nil {
		return fmt.Errorf("failed to get activity: %v", err)
	}

	// Add Device Info if available
	if source != nil {
		fitFile.FileId.ProductName = source.Name

		devInfo := fit.NewDeviceInfoMsg()
		devInfo.Timestamp = fitFile.FileId.TimeCreated
		devInfo.ProductName = source.Name

		activity.DeviceInfos = append(activity.DeviceInfos, devInfo)
	}

	// Map Activity Name to Sport/SubSport
	var sport fit.Sport
	var subSport fit.SubSport
	var sportName string

	switch strings.ToLower(activityName) {
	case "spinning":
		sport = fit.SportCycling
		subSport = fit.SubSportSpin
		sportName = "Spinning"
	case "bike", "cycling", "ride":
		sport = fit.SportCycling
		subSport = fit.SubSportGeneric
		sportName = "Cycling"
	case "run", "treadmill", "running":
		sport = fit.SportRunning
		// Since we filter GPS, likely treadmill or indoor
		subSport = fit.SubSportTreadmill
		sportName = "Treadmill Run"
	case "walk", "walking", "hike":
		sport = fit.SportWalking
		subSport = fit.SubSportGeneric
		sportName = "Walking"
	case "yoga":
		sport = fit.SportTraining
		subSport = fit.SubSportYoga
		sportName = "Yoga"
	case "elliptical":
		sport = fit.SportFitnessEquipment
		subSport = fit.SubSportElliptical
		sportName = "Elliptical"
	case "weights", "weight training", "strength training":
		sport = fit.SportTraining
		subSport = fit.SubSportStrengthTraining
		sportName = "Weight Training"
	case "workout":
		sport = fit.SportTraining
		subSport = fit.SubSportGeneric
		sportName = "Workout"
	default:
		// Default to generic training
		sport = fit.SportTraining
		subSport = fit.SubSportGeneric
		sportName = activityName
		if sportName == "" {
			sportName = "Workout"
		}
	}

	// Define Session
	session := &fit.SessionMsg{
		Sport:            sport,
		SubSport:         subSport,
		StartTime:        baseTime,
		Timestamp:        baseTime,
		TotalTimerTime:   0,
		SportProfileName: sportName,
	}

	if totalCalories > 0 {
		session.TotalCalories = uint16(totalCalories)
	}

	activity.Sessions = append(activity.Sessions, session)

	// Map Samples to Records
	count := 0
	var totalHR uint64
	var maxHR uint8
	var minHR uint8 = 255

	for _, s := range data.ActivitiesHeartIntraday.Dataset {
		sampleTime, err := time.Parse("15:04:05", s.Time)
		if err != nil {
			continue
		}

		actualTime := time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(),
			sampleTime.Hour(), sampleTime.Minute(), sampleTime.Second(), 0, time.Local)

		hr := uint8(s.Value)
		totalHR += uint64(hr)
		if hr > maxHR {
			maxHR = hr
		}
		if hr < minHR {
			minHR = hr
		}

		rec := &fit.RecordMsg{
			Timestamp: actualTime,
			HeartRate: hr,
			// Could also distribute calories here if we wanted to be precise, but session level is usually enough.
		}
		activity.Records = append(activity.Records, rec)
		count++
	}

	if count > 0 {
		// Update session timestamps based on first/last record
		session.StartTime = activity.Records[0].Timestamp
		session.Timestamp = activity.Records[count-1].Timestamp

		duration := session.Timestamp.Sub(session.StartTime)
		ms := uint32(duration.Milliseconds())

		session.TotalTimerTime = ms
		session.TotalElapsedTime = ms

		session.AvgHeartRate = uint8(totalHR / uint64(count))
		session.MaxHeartRate = maxHR
		session.MinHeartRate = minHR
	}

	// Finalize and Encode
	if err := fit.Encode(f, fitFile, binary.LittleEndian); err != nil {
		return fmt.Errorf("failed to encode fit file: %v", err)
	}

	return nil
}
