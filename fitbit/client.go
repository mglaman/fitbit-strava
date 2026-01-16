package fitbit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HeartRateResponse struct {
	ActivitiesHeartIntraday struct {
		Dataset []struct {
			Time  string  `json:"time"`
			Value float64 `json:"value"`
		} `json:"dataset"`
	} `json:"activities-heart-intraday"`
}

type ActivityLogSource struct {
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	ID              string   `json:"id"`
	URL             string   `json:"url"`
	TrackerFeatures []string `json:"trackerFeatures"`
}

type ActivityLog struct {
	LogID     int64             `json:"logId"`
	Name      string            `json:"activityName"`
	Calories  int               `json:"calories"`
	Duration  int               `json:"duration"`  // milliseconds
	StartTime string            `json:"startTime"` // ISO 8601
	Source    ActivityLogSource `json:"source"`
	HasGPS    bool              `json:"hasGps"`
}

type ActivityLogsResponse struct {
	Activities []ActivityLog `json:"activities"`
}

type Client struct {
	HttpClient *http.Client
}

func NewClient(client *http.Client) *Client {
	return &Client{HttpClient: client}
}

func (c *Client) FetchIntradayHeartRate(date, startTime, endTime string) (*HeartRateResponse, error) {
	url := fmt.Sprintf("https://api.fitbit.com/1/user/-/activities/heart/date/%s/1d/1sec/time/%s/%s.json",
		date, startTime, endTime)

	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fitbit api error: status %s", resp.Status)
	}

	var hrData HeartRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&hrData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &hrData, nil
}

func (c *Client) GetRecentActivities(limit int) (*ActivityLogsResponse, error) {
	// API requires exactly one of beforeDate or afterDate.
	// We use beforeDate=<tomorrow> to capture all recent activities including today's.
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	url := fmt.Sprintf("https://api.fitbit.com/1/user/-/activities/list.json?beforeDate=%s&sort=desc&offset=0&limit=%d",
		tomorrow, limit)
	// actually, the list endpoint is a bit tricky with afterDate/beforeDate for "recent".
	// Try the standard generic list endpoint or just use the activity log endpoint if it supports pagination?
	// https://dev.fitbit.com/build/reference/web-api/activity/get-activity-log-list/
	// GET https://api.fitbit.com/1/user/[user-id]/activities/list.json

	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent activities: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fitbit api error: status %s, body: %s", resp.Status, string(body))
	}

	var logs ActivityLogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("failed to decode recent activities: %v", err)
	}

	return &logs, nil
}

func (c *Client) GetActivityLogs(date string) (*ActivityLogsResponse, error) {
	url := fmt.Sprintf("https://api.fitbit.com/1/user/-/activities/date/%s.json", date)

	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch activity logs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fitbit api error: status %s", resp.Status)
	}

	var logs ActivityLogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("failed to decode activity logs: %v", err)
	}

	return &logs, nil
}
