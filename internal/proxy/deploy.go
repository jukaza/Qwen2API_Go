package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const vercelAPI = "https://api.vercel.com"

const relayFunctionCode = `
export const config = { runtime: "edge" };

export default async function handler(req) {
  const target = req.headers.get("x-relay-target");
  const relayPath = req.headers.get("x-relay-path") || "/";
  if (!target) {
    return new Response("LiteRouter Edge Relay Active", { status: 200 });
  }

  const targetUrl = target.replace(/\/$/, "") + relayPath;
  const headers = new Headers(req.headers);
  headers.delete("x-relay-target");
  headers.delete("x-relay-path");
  headers.delete("host");

  const response = await fetch(targetUrl, {
    method: req.method,
    headers,
    body: req.method !== "GET" && req.method !== "HEAD" ? req.body : undefined,
    duplex: "half",
  });

  return new Response(response.body, {
    status: response.status,
    headers: response.headers,
  });
}
`

type DeployVercelRequest struct {
	VercelToken string `json:"vercelToken"`
	ProjectName string `json:"projectName"`
	Count       int    `json:"count"`
}

type VercelDeployResult struct {
	DeployURL   string `json:"deployUrl"`
	ProjectName string `json:"projectName"`
}

func DeployVercel(ctx context.Context, req DeployVercelRequest) ([]VercelDeployResult, error) {
	if req.VercelToken == "" {
		return nil, fmt.Errorf("Vercel token is required")
	}

	projectNameInput := req.ProjectName
	if projectNameInput == "" {
		projectNameInput = fmt.Sprintf("relay-%d", time.Now().Unix())
	}
	count := req.Count
	if count < 1 {
		count = 1
	}
	if count > 10 {
		count = 10
	}

	var results []VercelDeployResult

	client := &http.Client{Timeout: 60 * time.Second}

	for i := 0; i < count; i++ {
		currentProjectName := projectNameInput
		if count > 1 {
			currentProjectName = fmt.Sprintf("%s-%d", projectNameInput, i+1)
		}

		payload := map[string]any{
			"name": currentProjectName,
			"files": []map[string]string{
				{
					"file": "api/relay.js",
					"data": relayFunctionCode,
				},
				{
					"file": "package.json",
					"data": fmt.Sprintf(`{"name": "%s", "version": "1.0.0"}`, currentProjectName),
				},
				{
					"file": "vercel.json",
					"data": `{"rewrites": [{"source": "/(.*)", "destination": "/api/relay"}]}`,
				},
			},
			"projectSettings": map[string]any{
				"framework": nil,
			},
			"target": "production",
		}

		bodyBytes, _ := json.Marshal(payload)
		deployReq, _ := http.NewRequestWithContext(ctx, "POST", vercelAPI+"/v13/deployments", bytes.NewReader(bodyBytes))
		deployReq.Header.Set("Authorization", "Bearer "+req.VercelToken)
		deployReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(deployReq)
		if err != nil {
			return results, fmt.Errorf("deploy error: %v", err)
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			return results, fmt.Errorf("failed to create Vercel deployment: %s", string(respBody))
		}

		var deployment struct {
			ID        string `json:"id"`
			UID       string `json:"uid"`
			ProjectID string `json:"projectId"`
			URL       string `json:"url"`
		}
		if err := json.Unmarshal(respBody, &deployment); err != nil {
			return results, fmt.Errorf("failed to parse deployment response: %v", err)
		}

		deploymentID := deployment.ID
		if deploymentID == "" {
			deploymentID = deployment.UID
		}
		projectID := deployment.ProjectID
		if projectID == "" {
			projectID = currentProjectName
		}

		// Disable SSO protection
		patchPayload := map[string]any{"ssoProtection": nil}
		patchBytes, _ := json.Marshal(patchPayload)
		patchReq, _ := http.NewRequestWithContext(ctx, "PATCH", vercelAPI+"/v9/projects/"+projectID, bytes.NewReader(patchBytes))
		patchReq.Header.Set("Authorization", "Bearer "+req.VercelToken)
		patchReq.Header.Set("Content-Type", "application/json")
		patchResp, err := client.Do(patchReq)
		if err == nil {
			patchResp.Body.Close()
		}

		// Poll until READY
		readyURL, err := pollVercelDeployment(ctx, client, deploymentID, req.VercelToken)
		if err != nil {
			return results, err
		}

		results = append(results, VercelDeployResult{
			DeployURL:   "https://" + readyURL,
			ProjectName: currentProjectName,
		})
	}

	return results, nil
}

func pollVercelDeployment(ctx context.Context, client *http.Client, deploymentID, token string) (string, error) {
	start := time.Now()
	for time.Since(start) < 2*time.Minute {
		req, _ := http.NewRequestWithContext(ctx, "GET", vercelAPI+"/v13/deployments/"+deploymentID, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var status struct {
			ReadyState string `json:"readyState"`
			URL        string `json:"url"`
		}
		json.Unmarshal(body, &status)

		if status.ReadyState == "READY" {
			return status.URL, nil
		}
		if status.ReadyState == "ERROR" || status.ReadyState == "CANCELED" {
			return "", fmt.Errorf("deployment failed with state %s", status.ReadyState)
		}

		time.Sleep(3 * time.Second)
	}

	return "", fmt.Errorf("deployment timed out")
}

func CleanVercelProjects(ctx context.Context, token string) (int, error) {
	if token == "" {
		return 0, fmt.Errorf("Vercel token is required")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", vercelAPI+"/v9/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to list projects: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("vercel api error: %s", string(body))
	}

	var data struct {
		Projects []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"projects"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return 0, fmt.Errorf("failed to parse projects: %w", err)
	}

	deletedCount := 0
	for _, p := range data.Projects {
		delReq, _ := http.NewRequestWithContext(ctx, "DELETE", vercelAPI+"/v9/projects/"+p.ID, nil)
		delReq.Header.Set("Authorization", "Bearer "+token)
		delResp, err := client.Do(delReq)
		if err == nil {
			delResp.Body.Close()
			if delResp.StatusCode < 400 {
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}
