package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultServerURL = "https://easylearn.baidu.com"
	UserAuthEndpoint = "http://127.0.0.1:8080/llm/test"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// AutoClaimResponse represents the response from starting auto claiming
type AutoClaimResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	TaskID  string `json:"taskId,omitempty"`
}

// Global variable to store the current auto claimer
var currentAutoClaimer *AutoClaimer

// StartAutoClaiming starts the auto claiming process
func (a *App) StartAutoClaiming(config AutoClaimConfig) AutoClaimResponse {
	// 设置默认服务器URL
	config.ServerBaseURL = DefaultServerURL
	log.Printf("StartAutoClaiming called with config: %+v", config)

	// 验证用户信息和LLM测试端点
	userName, err := a.validateUserAndLLM(config.Cookie)
	if err != nil {
		log.Printf("验证失败: %v", err)
		return AutoClaimResponse{
			Success: false,
			Message: fmt.Sprintf("验证失败: %v", err),
		}
	}

	log.Printf("用户验证成功: %s", userName)

	// Start auto claiming
	autoClaimer, err := StartAutoClaiming(a.ctx, config)
	if err != nil {
		log.Printf("Error starting auto claiming: %v", err)
		return AutoClaimResponse{
			Success: false,
			Message: fmt.Sprintf("启动自动认领失败: %v", err),
		}
	}

	// Store the auto claimer globally for later access
	currentAutoClaimer = autoClaimer

	log.Printf("Auto claiming started successfully")

	// Return success response
	return AutoClaimResponse{
		Success: true,
		Message: "自动认领已启动",
		TaskID:  "task_" + fmt.Sprintf("%d", config.ClaimLimit), // Simple task ID generation
	}
}

// GetTaskLabels 获取任务标签数据
func (a *App) GetTaskLabels(taskType, cookie string) (map[string]interface{}, error) {
	response, err := GetAuditTaskLabel(taskType, DefaultServerURL, cookie)
	if err != nil {
		return nil, err
	}

	// 转换为 map 以避免 TypeScript 生成器问题
	result := map[string]interface{}{
		"errno":  response.Errno,
		"errmsg": response.Errmsg,
		"data": map[string]interface{}{
			"filter": response.Data.Filter,
		},
	}
	return result, nil
}

// StopAutoClaiming 停止自动认领过程
func (a *App) StopAutoClaiming() AutoClaimResponse {
	if currentAutoClaimer == nil {
		return AutoClaimResponse{
			Success: false,
			Message: "没有运行的自动认领任务",
		}
	}

	currentAutoClaimer.Stop()

	return AutoClaimResponse{
		Success: true,
		Message: "自动认领已停止",
	}
}

// AutoClaimStatusResponse 表示自动认领状态响应
type AutoClaimStatusResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	IsActive         bool   `json:"isActive"`
	SuccessfulClaims int    `json:"successfulClaims"`
	LastError        string `json:"lastError"`
}

// GetAutoClaimStatus 获取自动认领状态
func (a *App) GetAutoClaimStatus() AutoClaimStatusResponse {
	if currentAutoClaimer == nil {
		return AutoClaimStatusResponse{
			Success:          true,
			Message:          "无运行任务",
			IsActive:         false,
			SuccessfulClaims: 0,
			LastError:        "",
		}
	}

	status := currentAutoClaimer.GetStatus()

	return AutoClaimStatusResponse{
		Success:          true,
		Message:          "状态获取成功",
		IsActive:         status.IsActive,
		SuccessfulClaims: status.SuccessfulClaims,
		LastError:        status.LastError,
	}
}

// GetUserInfo 获取用户信息
func (a *App) GetUserInfo(cookie string) (map[string]interface{}, error) {
	response, err := GetUserInfo(DefaultServerURL, cookie)
	if err != nil {
		return nil, err
	}

	// 转换为 map 以避免 TypeScript 生成器问题
	result := map[string]interface{}{
		"errno":  response.Errno,
		"errmsg": response.Errmsg,
		"data": map[string]interface{}{
			"roleLinks": response.Data.RoleLinks,
			"roleNames": response.Data.RoleNames,
			"userName":  response.Data.UserName,
			"avatar":    response.Data.Avatar,
		},
	}
	return result, nil
}

// validateUserAndLLM 验证用户信息和LLM测试端点
func (a *App) validateUserAndLLM(cookie string) (string, error) {
	// 获取用户信息
	userInfo, err := GetUserInfo(DefaultServerURL, cookie)
	if err != nil {
		return "", fmt.Errorf("无权使用该软件，请联系管理员")
	}

	if userInfo == nil || userInfo.Errno != 0 {
		return "", fmt.Errorf("无权使用该软件，请联系管理员")
	}

	userName := userInfo.Data.UserName
	if userName == "" {
		return "", fmt.Errorf("无权使用该软件，请联系管理员")
	}

	// 验证LLM测试端点
	encodedUserName := url.QueryEscape(userName)

	req, err := http.NewRequest("GET", UserAuthEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置认证头部，使用URL编码后的用户名
	req.Header.Set("Authorization", encodedUserName)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("连接LLM测试端点失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM测试端点返回错误: HTTP %d", resp.StatusCode)
	}

	var responseData map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应格式
	if responseData["text"] != nil {
		text := responseData["text"].(string)
		if text != "ok" && text != "" {
			return "", fmt.Errorf("无权使用该软件，请联系管理员")
		}
	}

	// 检查响应格式 {"error": "ok"}
	if responseData["error"] != nil {
		errorMsg := responseData["error"].(string)
		if errorMsg != "ok" {
			return "", fmt.Errorf("无权使用该软件，请联系管理员")
		}
	}

	// 如果没有预期的字段，也视为失败
	if responseData["text"] == nil && responseData["error"] == nil {
		return "", fmt.Errorf("无权使用该软件，请联系管理员")
	}

	return userName, nil
}
