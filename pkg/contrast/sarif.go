package contrast

import (
	"io"
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

func (c *Client) StartAsyncSarifGeneration(appUuid string) (string, error) {
    url := fmt.Sprintf("%s/Contrast/api/ng/organizations/%s/applications/%s/sarif/async",
        c.BaseURL, c.OrgID, appUuid)
    
    payload := map[string]interface{}{
    "severities": []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "NOTE"},
    "quickFilter": "OPEN",
    }
    bodyBytes, _ := json.Marshal(payload)
    body := bytes.NewBuffer(bodyBytes) 
    req, err := http.NewRequest("POST", url, body)
    if err != nil {
        return "", fmt.Errorf("failed to create request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json") 
    c.addAuth(req)

    resp, err := c.HttpClient.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to call Contrast API: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    var sarifResp struct {
        Messages []string `json:"messages"`
        Success  bool     `json:"success"`
        Uuid     string   `json:"uuid"`  
    }
    if err := json.NewDecoder(resp.Body).Decode(&sarifResp); err != nil {
        return "", fmt.Errorf("failed to parse response: %w", err)
    }

    if !sarifResp.Success || sarifResp.Uuid == "" {
        return "", fmt.Errorf("SARIF generation request failed: %v", sarifResp.Messages)
    }

    return sarifResp.Uuid, nil
}


func (c *Client) PollSarifGenerationStatus(sarifUuid string) (string, error) {
    url := fmt.Sprintf("%s/Contrast/api/ng/organizations/%s/reports/%s/status", c.BaseURL, c.OrgID, sarifUuid)

    var statusResp struct {
        Messages    []string `json:"messages"`
        Success     bool     `json:"success"`
        Status      string   `json:"status"`
        DownloadUrl string   `json:"downloadUrl"`
    }

    maxTotalWait := 5 * time.Minute        // total timeout
    maxPollInterval := 60 * time.Second    // maximum between requests
    initialDelay := 15 * time.Second       // delay before first poll
    pollInterval := 5 * time.Second        // starting interval after initialDelay
    backoffFactor := 1.5                   // interval increase factor

    totalWaited := time.Duration(0)

    time.Sleep(initialDelay)
    totalWaited += initialDelay

    for totalWaited < maxTotalWait {
        req, err := http.NewRequest("GET", url, nil)
        if err != nil {
            return "", fmt.Errorf("failed to create request: %w", err)
        }
        c.addAuth(req)

        resp, err := c.HttpClient.Do(req)
        if err != nil {
            return "", fmt.Errorf("failed to call Contrast API: %w", err)
        }

        if resp.StatusCode != http.StatusOK {
            resp.Body.Close()
            return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
        }

        if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
            resp.Body.Close()
            return "", fmt.Errorf("failed to parse response: %w", err)
        }
        resp.Body.Close()

        if !statusResp.Success {
            return "", fmt.Errorf("SARIF status check failed: %v", statusResp.Messages)
        }

        if statusResp.Status == "ACTIVE" && statusResp.DownloadUrl != "" {
            return statusResp.DownloadUrl, nil
        }

        if statusResp.Status != "CREATING" {
            return "", fmt.Errorf("unexpected SARIF status: %s", statusResp.Status)
        }

        time.Sleep(pollInterval)
        totalWaited += pollInterval

        nextInterval := time.Duration(float64(pollInterval) * backoffFactor)
        if nextInterval > maxPollInterval {
            pollInterval = maxPollInterval
        } else {
            pollInterval = nextInterval
        }
    }

    return "", fmt.Errorf("SARIF generation timed out after waiting %s", maxTotalWait)
}

func (c *Client) DownloadSarif(downloadUrl string) ([]byte, error) {
    req, err := http.NewRequest("POST", downloadUrl, nil)
    if err != nil {
        return nil, err
    }
    c.addAuth(req)

    resp, err := c.HttpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    return io.ReadAll(resp.Body)
}
