package contrast

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStartAsyncSarifGeneration_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "uuid": "abc-123"}`))
	}))
	defer server.Close()

	client := &Client{
		OrgID:      "org-1",
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}

	uuid, err := client.StartAsyncSarifGeneration("app-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if uuid != "abc-123" {
		t.Errorf("Expected uuid 'abc-123', got '%s'", uuid)
	}
}

func TestStartAsyncSarifGeneration_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		OrgID:      "org-1",
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}

	_, err := client.StartAsyncSarifGeneration("app-1")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

func TestPollSarifGenerationStatus_Success(t *testing.T) {
	counter := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter++
		if counter == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "status": "CREATING"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true, "status": "ACTIVE", "downloadUrl": "http://example.com/sarif"}`))
		}
	}))
	defer server.Close()

	client := &Client{
		OrgID:      "org-1",
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}

	downloadUrl, err := client.PollSarifGenerationStatus("uuid-1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if downloadUrl != "http://example.com/sarif" {
		t.Errorf("Expected downloadUrl, got '%s'", downloadUrl)
	}
}

func TestPollSarifGenerationStatus_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "status": "CREATING"}`))
	}))
	defer server.Close()

	client := &Client{
		OrgID:      "org-1",
		BaseURL:    server.URL,
		HttpClient: server.Client(),
	}

	_, err := client.PollSarifGenerationStatus("uuid-1")
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

func TestDownloadSarif_Success(t *testing.T) {
	expectedContent := "sarif content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	client := &Client{
		HttpClient: server.Client(),
	}

	data, err := client.DownloadSarif(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(data) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(data))
	}
}

func TestDownloadSarif_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := &Client{
		HttpClient: server.Client(),
	}

	_, err := client.DownloadSarif(server.URL)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}
