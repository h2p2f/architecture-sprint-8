package test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type ReportResponse struct {
	Reports []struct {
		ID       string  `json:"id"`
		UserID   string  `json:"userId"`
		Value    float64 `json:"value"`
		Category string  `json:"category"`
	} `json:"reports"`
	Total int `json:"total"`
}

func TestAPIFlow(t *testing.T) {
	// Получаем токен
	tokenResp, err := getToken("prothetic1", "prothetic123")
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	// Проверяем наличие токена
	if tokenResp.AccessToken == "" {
		t.Fatal("Received empty access token")
	}

	// Логируем для отладки
	t.Logf("Received token type: %s", tokenResp.TokenType)

	// Тестируем API
	reports, statusCode, err := getReports(tokenResp.AccessToken)
	if err != nil {
		t.Fatalf("Failed to get reports: %v", err)
	}

	if statusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", statusCode)
	}

	if reports == nil {
		t.Fatal("No response received from API")
	}
}

func getToken(username, password string) (*TokenResponse, error) {
	tokenURL := "http://localhost:8080/realms/reports-realm/protocol/openid-connect/token"
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", "reports-frontend")
	data.Set("username", username)
	data.Set("password", password)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v\nBody: %s", err, string(body))
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("auth error: %s - %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	// Логируем успешное получение токена
	log.Printf("Successfully received token of type: %s", tokenResp.TokenType)

	return &tokenResp, nil
}

func getReports(token string) (*ReportResponse, int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:8000/reports", nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	// Добавляем Bearer префикс, если его нет
	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}

	req.Header.Set("Authorization", token)

	// Логируем отправляемый запрос
	log.Printf("Sending request to %s with token: %s", req.URL, token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %v", err)
	}

	// Логируем ответ
	log.Printf("Received response with status %d: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var reportResp ReportResponse
	if err := json.Unmarshal(body, &reportResp); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to parse response: %v\nBody: %s", err, string(body))
	}

	return &reportResp, resp.StatusCode, nil
}
