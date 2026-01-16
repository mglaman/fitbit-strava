package strava

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type ActivityMetadata struct {
	Name        string
	Description string
	ExternalID  string
}

type Client struct {
	HttpClient *http.Client
}

func NewClient(client *http.Client) *Client {
	return &Client{HttpClient: client}
}

func (c *Client) UploadActivity(filename string, metadata ActivityMetadata) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}
	io.Copy(part, file)

	// Add fields
	writer.WriteField("data_type", "fit")
	if metadata.Name != "" {
		writer.WriteField("name", metadata.Name)
	}
	if metadata.Description != "" {
		writer.WriteField("description", metadata.Description)
	}
	if metadata.ExternalID != "" {
		writer.WriteField("external_id", metadata.ExternalID)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close writer: %v", err)
	}

	req, err := http.NewRequest("POST", "https://www.strava.com/api/v3/uploads", body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("strava api error: status %s, body %s", resp.Status, string(respBody))
	}

	return string(respBody), nil
}
