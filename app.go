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
	DefaultServerURL    = "https://easylearn.baidu.com"
	UserAuthEndpoint    = "http://127.0.0.1:8080/llm/test"
	PocketBaseURL       = "http://47.109.61.89:5913"
	BackupPocketBaseURL = "https://pb.pingfury.top"
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

	// 根据授权类型进行验证
	switch config.AuthType {
	case "official":
		// 官方授权：验证用户名
		if config.AuthUsername == "" {
			log.Printf("官方授权验证失败: 用户名不能为空")
			return AutoClaimResponse{
				Success: false,
				Message: "官方授权用户名不能为空",
			}
		}

		err := a.validateOfficialAuth(config.AuthUsername)
		if err != nil {
			log.Printf("官方授权验证失败: %v", err)
			return AutoClaimResponse{
				Success: false,
				Message: fmt.Sprintf("官方授权验证失败: %v", err),
			}
		}

		log.Printf("官方授权验证成功: %s", config.AuthUsername)

	case "custom":
		// 定制授权：使用原有的LLM测试端点验证
		userName, err := a.validateUserAndLLM(config.Cookie)
		if err != nil {
			log.Printf("定制授权验证失败: %v", err)
			return AutoClaimResponse{
				Success: false,
				Message: fmt.Sprintf("验证失败: %v", err),
			}
		}

		log.Printf("定制授权验证成功: %s", userName)

	default:
		log.Printf("未知的授权类型: %s", config.AuthType)
		return AutoClaimResponse{
			Success: false,
			Message: "未知的授权类型",
		}
	}

	// Start auto claiming
	if config.Interval < 1 {
		log.Printf("传递给StartAutoClaiming的Interval值为: %.3f秒 (%.0f毫秒)", config.Interval, config.Interval*1000)
	} else {
		log.Printf("传递给StartAutoClaiming的Interval值为: %.1f秒", config.Interval)
	}
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
func (a *App) GetTaskLabels(taskType, cookie string) (map[string]any, error) {
	response, err := GetAuditTaskLabel(taskType, DefaultServerURL, cookie)
	if err != nil {
		return nil, err
	}

	// 转换为 map 以避免 TypeScript 生成器问题
	result := map[string]any{
		"errno":  response.Errno,
		"errmsg": response.Errmsg,
		"data": map[string]any{
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
func (a *App) GetUserInfo(cookie string) (map[string]any, error) {
	response, err := GetUserInfo(DefaultServerURL, cookie)
	if err != nil {
		return nil, err
	}

	// 转换为 map 以避免 TypeScript 生成器问题
	result := map[string]any{
		"errno":  response.Errno,
		"errmsg": response.Errmsg,
		"data": map[string]any{
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

	var responseData map[string]any
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

// BaiduEduUserResponse 表示百度教育用户API响应
type BaiduEduUserResponse struct {
	ID             string    `json:"id"`
	CollectionID   string    `json:"collectionId"`
	CollectionName string    `json:"collectionName"`
	CreatedRaw     string    `json:"created"`
	UpdatedRaw     string    `json:"updated"`
	Name           string    `json:"name"`
	ExpTimeRaw     string    `json:"exp_time"`
	Created        time.Time `json:"-"`
	Updated        time.Time `json:"-"`
	ExpTime        time.Time `json:"-"`
	Remark         string    `json:"remark"`
	Limit          int       `json:"limit"`
	XufeiType      string    `json:"xufei_type"`
	Coze           bool      `json:"coze"`
}

// validateOfficialAuth 验证官方授权用户
func (a *App) validateOfficialAuth(username string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}

	// 尝试主服务器和备用服务器
	var lastErr error
	var userResponse BaiduEduUserResponse
	servers := []string{PocketBaseURL, BackupPocketBaseURL}
	var success bool

	for i, serverURL := range servers {
		// 构建API URL
		apiURL := fmt.Sprintf("%s/api/collections/baidu_edu_users/records/%s", serverURL, username)

		// 创建HTTP请求
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("创建请求失败: %v", err)
			continue
		}

		// 设置请求头
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		// 发送请求，设置较短的超时时间
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("请求用户信息失败: %v", err)
			if i < len(servers)-1 {
				log.Printf("主服务器连接失败，尝试备用服务器: %v", err)
				continue
			}
			return lastErr
		}
		defer resp.Body.Close()

		// 检查HTTP状态码
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("用户不存在或无权使用")
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API请求失败: HTTP %d", resp.StatusCode)
			if i < len(servers)-1 {
				log.Printf("主服务器返回错误，尝试备用服务器: HTTP %d", resp.StatusCode)
				continue
			}
			return lastErr
		}

		// 读取响应体
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("读取响应失败: %v", err)
			if i < len(servers)-1 {
				log.Printf("读取响应失败，尝试备用服务器: %v", err)
				continue
			}
			return lastErr
		}

		// 解析JSON响应
		err = json.Unmarshal(body, &userResponse)
		if err != nil {
			lastErr = fmt.Errorf("解析响应失败: %v", err)
			if i < len(servers)-1 {
				log.Printf("解析响应失败，尝试备用服务器: %v", err)
				continue
			}
			return lastErr
		}

		// 成功获取响应，跳出循环
		log.Printf("成功连接到服务器: %s", serverURL)
		success = true
		break
	}

	// 如果所有服务器都尝试失败
	if !success {
		return lastErr
	}

	// PocketBase时间解析函数
	parsePocketBaseTime := func(timeStr string) (time.Time, error) {
		// PocketBase返回的格式: "2025-01-07 11:06:37.080Z"
		return time.Parse("2006-01-02 15:04:05.000Z", timeStr)
	}

	// 解析过期时间
	expTime, err := parsePocketBaseTime(userResponse.ExpTimeRaw)
	if err != nil {
		return fmt.Errorf("解析过期时间失败: %v", err)
	}
	userResponse.ExpTime = expTime

	// 同时解析其他时间字段用于日志记录
	if userResponse.CreatedRaw != "" {
		if createdTime, err := parsePocketBaseTime(userResponse.CreatedRaw); err == nil {
			userResponse.Created = createdTime
		}
	}
	if userResponse.UpdatedRaw != "" {
		if updatedTime, err := parsePocketBaseTime(userResponse.UpdatedRaw); err == nil {
			userResponse.Updated = updatedTime
		}
	}

	// 检查过期时间
	now := time.Now()
	if now.After(userResponse.ExpTime) {
		return fmt.Errorf("授权已过期，过期时间: %s", userResponse.ExpTime.Format("2006-01-02 15:04:05"))
	}

	// 检查是否在有效期前1小时内（可选警告）
	if now.Add(time.Hour).After(userResponse.ExpTime) {
		log.Printf("警告：用户授权即将过期，过期时间: %s", userResponse.ExpTime.Format("2006-01-02 15:04:05"))
	}

	log.Printf("官方授权验证成功: 用户 %s, 过期时间: %s", userResponse.Name, userResponse.ExpTime.Format("2006-01-02 15:04:05"))
	return nil
}
