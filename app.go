package main

import (
	"context"
	"fmt"
	"log"
)

const DefaultServerURL = "https://easylearn.baidu.com"

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
