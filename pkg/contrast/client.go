package contrast

import (
    "net/http"
    "time"
)

type Client struct {
    ApiKey     string
    ServiceKey string
    Username   string
    OrgID      string
    BaseURL    string
    HttpClient *http.Client
}

func NewClient(apiKey, serviceKey, username, orgID, baseURL string) *Client {
    if baseURL == "" {
    baseURL = "https://cs003.contrastsecurity.com"
    }
    return &Client{
        ApiKey:     apiKey,
        ServiceKey: serviceKey,
        Username:   username,
        OrgID:      orgID,
        BaseURL:    baseURL,
        HttpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) addAuth(req *http.Request) {
        req.SetBasicAuth(c.Username, c.ServiceKey)
        req.Header.Set("API-Key", c.ApiKey)
}
