package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AutoClaimConfig 保存自动认领过程的所有配置参数
type AutoClaimConfig struct {
	// 必需参数
	ServerBaseURL string  // 服务器的基础 URL
	Cookie        string  // 认证 cookie
	TaskType      string  // 要认领的任务类型（"audittask" 或 "producetask"）
	ClaimLimit    int     // 要认领的最大任务数
	Interval      float64 // 认领尝试之间的间隔（秒），支持小数，最小 0.1 秒

	// 随机页面参数
	MaxPages int // 请求时的最大随机页码，0 表示禁用随机页码（始终请求第1页）

	// 并发认领参数
	ConcurrentClaims int // 并发认领的任务数量，默认为10个

	// 筛选参数
	StepID     int // 学段 ID
	SubjectID  int // 学科 ID
	ClueTypeID int // 线索类型 ID

	// 关键词过滤器
	IncludeKeywords []string // 任务简介中必须存在的关键词
	ExcludeKeywords []string // 任务简介中不能存在的关键词

	// 发布时间过滤器
	StartTime string // 开始时间，格式: "2006-01-02 15:04:05"
	EndTime   string // 结束时间，格式: "2006-01-02 15:04:05"
}

// ClaimStatus 表示自动认领过程的当前状态
type ClaimStatus struct {
	SuccessfulClaims int            // 成功认领的任务数
	LastError        string         // 最后的错误消息（如果有）
	IsActive         bool           // 自动认领过程是否处于活动状态
	LastResponse     *ClaimResponse // 来自认领 API 的最后响应
}

// AutoClaimer 处理任务的自动认领
type AutoClaimer struct {
	config       AutoClaimConfig
	status       ClaimStatus
	cancel       context.CancelFunc
	mutex        sync.RWMutex
	actualClaims int
	attemptCount int
	logCh        chan string // 用于非阻塞日志记录的通道
}

// filterByKeywords 根据包含和排除关键词筛选任务
func filterByKeywords(text string, includeKeywords, excludeKeywords []string) bool {
	text = strings.ToLower(text)

	// Check exclude keywords first (faster rejection)
	for _, keyword := range excludeKeywords {
		if keyword != "" && strings.Contains(text, strings.ToLower(keyword)) {
			return false
		}
	}

	// If no include keywords are specified, accept the task
	if len(includeKeywords) == 0 {
		return true
	}

	// Check if any include keyword is present
	for _, keyword := range includeKeywords {
		if keyword != "" && strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}

	// If include keywords are specified but none match, reject the task
	return false
}

// filterByDispatchTime 根据发布时间筛选任务
func filterByDispatchTime(dispatchTime, startTime, endTime string) bool {
	// 如果没有设置时间过滤器，接受所有任务
	if startTime == "" && endTime == "" {
		return true
	}

	// 如果任务没有dispatchTime，拒绝
	if dispatchTime == "" {
		return false
	}

	// 解析任务的发布时间
	taskTime, err := time.Parse("2006-01-02 15:04:05", dispatchTime)
	if err != nil {
		// 如果时间格式不正确，拒绝
		return false
	}

	// 检查开始时间约束
	if startTime != "" {
		startTimeObj, err := time.Parse("2006-01-02 15:04:05", startTime)
		if err != nil {
			// 如果开始时间格式不正确，跳过开始时间检查
		} else if taskTime.Before(startTimeObj) {
			return false
		}
	}

	// 检查结束时间约束
	if endTime != "" {
		endTimeObj, err := time.Parse("2006-01-02 15:04:05", endTime)
		if err != nil {
			// 如果结束时间格式不正确，跳过结束时间检查
		} else if taskTime.After(endTimeObj) {
			return false
		}
	}

	return true
}

// NewAutoClaimer 使用给定的配置创建一个新的 AutoClaimer
func NewAutoClaimer(config AutoClaimConfig) *AutoClaimer {
	// Set default values if not provided
	if config.TaskType == "" {
		config.TaskType = "audittask"
	}

	if config.Interval < 0.1 {
		config.Interval = 1.0
	}

	if config.ClaimLimit <= 0 {
		config.ClaimLimit = 10
	}

	if config.MaxPages < 0 {
		config.MaxPages = 0
	}

	if config.ConcurrentClaims <= 0 {
		config.ConcurrentClaims = 10
	}

	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	return &AutoClaimer{
		config: config,
		status: ClaimStatus{
			IsActive: false,
		},
		logCh: make(chan string, 100), // 创建带缓冲的日志通道，避免阻塞
	}
}

// GetStatus 返回自动认领过程的当前状态
func (ac *AutoClaimer) GetStatus() ClaimStatus {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()
	return ac.status
}

// Start 开始自动认领过程
func (ac *AutoClaimer) Start(ctx context.Context) error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// 检查是否已经在运行
	if ac.status.IsActive {
		return fmt.Errorf("auto-claiming is already active")
	}

	// 重置计数器
	ac.actualClaims = 0
	ac.attemptCount = 0
	ac.status.SuccessfulClaims = 0
	ac.status.LastError = ""
	ac.status.LastResponse = nil

	// 创建一个带有取消功能的新上下文
	ctxWithCancel, cancel := context.WithCancel(ctx)
	ac.cancel = cancel
	ac.status.IsActive = true

	// 日志处理由调用者处理

	// 在一个 goroutine 中启动自动认领循环
	go ac.autoClaimLoop(ctxWithCancel)

	return nil
}

// Stop 停止自动认领过程
func (ac *AutoClaimer) Stop() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	if ac.status.IsActive && ac.cancel != nil {
		ac.cancel()
		ac.status.IsActive = false
		ac.cancel = nil
	}
}

// autoClaimLoop 是自动认领过程的主循环
func (ac *AutoClaimer) autoClaimLoop(ctx context.Context) {
	// 立即执行初始认领尝试
	ac.performAutoClaiming(ctx)

	// 设置定时器进行周期性认领尝试
	ticker := time.NewTicker(time.Duration(ac.config.Interval * float64(time.Second)))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context was cancelled, exit the loop
			select {
			case ac.logCh <- fmt.Sprintf("[%s] 由于上下文取消，自动认领已停止", time.Now().Format("2006-01-02 15:04:05")):
				// 消息已发送到通道
			default:
				// 通道已满，但我们不想阻塞，所以忽略
			}
			return
		case <-ticker.C:
			// Time to attempt another claim
			ac.performAutoClaiming(ctx)
		}
	}
}

// performAutoClaiming 尝试根据配置认领任务
func (ac *AutoClaimer) performAutoClaiming(ctx context.Context) {
	// Check if context is already cancelled
	if ctx.Err() != nil {
		return
	}

	ac.mutex.Lock()
	ac.attemptCount++
	attemptNum := ac.attemptCount
	actualClaims := ac.actualClaims
	ac.mutex.Unlock()

	// 使用非阻塞方式发送日志消息
	select {
	case ac.logCh <- fmt.Sprintf("[%s] 认领尝试 #%d 开始，当前认领数：%d/%d", time.Now().Format("2006-01-02 15:04:05"), attemptNum, actualClaims, ac.config.ClaimLimit):
		// 消息已发送到通道
	default:
		// 通道已满，但我们不想阻塞，所以忽略
	}

	// Check if we've reached the claim limit
	if actualClaims >= ac.config.ClaimLimit {
		// 使用非阻塞方式发送日志消息
		select {
		case ac.logCh <- fmt.Sprintf("[%s] 认领限制已达到 (%d/%d)，停止自动认领", time.Now().Format("2006-01-02 15:04:05"), actualClaims, ac.config.ClaimLimit):
			// 消息已发送到通道
		default:
			// 通道已满，但我们不想阻塞，所以忽略
		}
		ac.Stop()
		return
	}

	// 计算还需要认领多少个任务
	remainingClaimsNeeded := ac.config.ClaimLimit - actualClaims

	// 确定页码
	pageNum := 1
	if ac.config.MaxPages > 1 {
		// 使用随机页码，范围从 1 到 MaxPages
		pageNum = rand.Intn(ac.config.MaxPages) + 1
		// 记录使用的随机页码
		select {
		case ac.logCh <- fmt.Sprintf("[%s] 使用随机页码：第 %d 页（共 %d 页）", time.Now().Format("2006-01-02 15:04:05"), pageNum, ac.config.MaxPages):
		default:
		}
	}

	// 尝试获取任务列表
	options := map[string]interface{}{
		"pn":       pageNum,
		"rn":       20,
		"clueID":   "",
		"clueType": ac.config.ClueTypeID,
		"step":     ac.config.StepID,
		"subject":  ac.config.SubjectID,
		"taskType": ac.config.TaskType,
	}

	// 直接使用配置中的服务器URL和Cookie
	serverBaseURL := ac.config.ServerBaseURL
	cookie := ac.config.Cookie

	// 获取任务列表
	res, err := GetAuditTaskList(options, serverBaseURL, cookie)
	if err != nil {
		ac.setError(fmt.Sprintf("获取任务列表出错：%v", err))
		return
	}

	// 检查请求是否成功
	if res.Errno != 0 || res.Data.List == nil {
		ac.setError(fmt.Sprintf("获取任务列表失败：%s", res.Errmsg))
		return
	}

	// 根据关键词和发布时间筛选任务
	var filteredTasks []TaskItem
	for _, task := range res.Data.List {
		textToCheck := task.Brief
		// 首先检查关键词过滤
		if !filterByKeywords(textToCheck, ac.config.IncludeKeywords, ac.config.ExcludeKeywords) {
			continue
		}
		// 只有生产任务才检查发布时间过滤
		if ac.config.TaskType == "producetask" {
			if !filterByDispatchTime(task.DispatchTime, ac.config.StartTime, ac.config.EndTime) {
				continue
			}
		}
		// 条件满足，添加到筛选结果
		filteredTasks = append(filteredTasks, task)
	}

	// 使用非阻塞方式发送日志消息
	var filterMsg string
	if ac.config.TaskType == "producetask" && (ac.config.StartTime != "" || ac.config.EndTime != "") {
		filterMsg = fmt.Sprintf("（关键词+时间筛选）")
	} else {
		filterMsg = fmt.Sprintf("（关键词筛选）")
	}
	select {
	case ac.logCh <- fmt.Sprintf("[%s] 已筛选任务：%d/%d %s", time.Now().Format("2006-01-02 15:04:05"), len(filteredTasks), len(res.Data.List), filterMsg):
		// 消息已发送到通道
	default:
		// 通道已满，但我们不想阻塞，所以忽略
	}

	// 检查是否有任务可认领
	if len(filteredTasks) == 0 {
		ac.setError("线索池中没任务")
		return
	}

	// 将要认领的任务数量限制为我们所需的数量
	if len(filteredTasks) > remainingClaimsNeeded {
		filteredTasks = filteredTasks[:remainingClaimsNeeded]
	}

	// 根据任务类型提取任务 ID
	var taskIDs []string
	for _, task := range filteredTasks {
		if ac.config.TaskType == "producetask" {
			taskIDs = append(taskIDs, strconv.Itoa(task.ClueID))
		} else {
			taskIDs = append(taskIDs, strconv.Itoa(task.TaskID))
		}
	}

	// 使用非阻塞方式发送日志消息
	select {
	case ac.logCh <- fmt.Sprintf("[%s] 尝试并发认领 %d 个任务（并发数：%d）", time.Now().Format("2006-01-02 15:04:05"), len(taskIDs), ac.config.ConcurrentClaims):
		// 消息已发送到通道
	default:
		// 通道已满，但我们不想阻塞，所以忽略
	}

	// 并发认领任务
	var wg sync.WaitGroup
	var mu sync.Mutex
	var lastClaimRes *ClaimResponse
	var lastErr error
	successCount := 0

	// 创建任务通道用于并发处理
	taskChan := make(chan string, len(taskIDs))
	for _, taskID := range taskIDs {
		taskChan <- taskID
	}
	close(taskChan)

	// 启动并发工作池
	concurrentClaims := ac.config.ConcurrentClaims
	if concurrentClaims > len(taskIDs) {
		concurrentClaims = len(taskIDs)
	}

	for i := 0; i < concurrentClaims; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for taskID := range taskChan {
				// 认领单个任务
				claimRes, err := ClaimAuditTask([]string{taskID}, ac.config.TaskType, serverBaseURL, cookie)

				mu.Lock()
				if err != nil {
					lastErr = err
					mu.Unlock()
					continue
				}

				lastClaimRes = claimRes

				// 检查认领是否成功
				if claimRes.Errno == 0 {
					// 尝试提取成功认领的任务数
					taskSuccessCount := 0

					// 处理不同的响应格式
					switch data := claimRes.Data.(type) {
					case map[string]interface{}:
						if success, ok := data["success"].(float64); ok {
							taskSuccessCount = int(success)
						}
					case struct{ Success int }:
						taskSuccessCount = data.Success
					}

					successCount += taskSuccessCount
				}
				mu.Unlock()
			}
		}()
	}

	// 等待所有并发任务完成
	wg.Wait()

	// 如果所有任务都失败了，设置错误
	if lastErr != nil && successCount == 0 {
		ac.setError(fmt.Sprintf("认领任务出错：%v", lastErr))
		return
	}

	// 使用最后一个响应作为整体响应
	claimRes := lastClaimRes
	if claimRes == nil {
		ac.setError("没有有效的认领响应")
		return
	}

	// 用认领响应更新状态
	ac.mutex.Lock()
	ac.status.LastResponse = claimRes

	// 更新认领计数（successCount 已在循环中计算）
	ac.actualClaims += successCount
	ac.status.SuccessfulClaims = ac.actualClaims
	ac.status.LastError = ""

	// 使用非阻塞方式发送日志消息，包含认领的任务ID
	idsStr := strings.Join(taskIDs, ", ")
	var logMessage string
	if ac.config.TaskType == "producetask" {
		logMessage = fmt.Sprintf("[%s] 并发认领完成：成功认领 %d 个任务（并发数：%d），ClueID: [%s]，总计：%d/%d", time.Now().Format("2006-01-02 15:04:05"), successCount, ac.config.ConcurrentClaims, idsStr, ac.actualClaims, ac.config.ClaimLimit)
	} else {
		logMessage = fmt.Sprintf("[%s] 并发认领完成：成功认领 %d 个任务（并发数：%d），TaskID: [%s]，总计：%d/%d", time.Now().Format("2006-01-02 15:04:05"), successCount, ac.config.ConcurrentClaims, idsStr, ac.actualClaims, ac.config.ClaimLimit)
	}
	select {
	case ac.logCh <- logMessage:
		// 消息已发送到通道
	default:
		// 通道已满，但我们不想阻塞，所以忽略
	}

	// Check if we've reached the claim limit
	if ac.actualClaims >= ac.config.ClaimLimit {
		// 使用非阻塞方式发送日志消息
		select {
		case ac.logCh <- fmt.Sprintf("[%s] 认领限制已达到（%d/%d），停止自动认领", time.Now().Format("2006-01-02 15:04:05"), ac.actualClaims, ac.config.ClaimLimit):
			// 消息已发送到通道
		default:
			// 通道已满，但我们不想阻塞，所以忽略
		}
		ac.status.IsActive = false
		if ac.cancel != nil {
			ac.cancel()
			ac.cancel = nil
		}
	}

	ac.mutex.Unlock()
}

// setError 更新错误状态
func (ac *AutoClaimer) setError(errMsg string) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	ac.status.LastError = errMsg
	// 使用非阻塞方式发送日志消息
	select {
	case ac.logCh <- fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), errMsg):
		// 消息已发送到通道
	default:
		// 通道已满，但我们不想阻塞，所以忽略
	}
}

// StartAutoClaiming 是一个便捷函数，用于创建并启动 AutoClaimer
func StartAutoClaiming(ctx context.Context, config AutoClaimConfig) (*AutoClaimer, error) {
	// 验证必需参数
	if config.ServerBaseURL == "" {
		return nil, fmt.Errorf("server base URL is required")
	}

	if config.Cookie == "" {
		return nil, fmt.Errorf("cookie is required")
	}

	// 创建自动认领器
	autoClaimer := NewAutoClaimer(config)

	// 启动自动认领过程
	err := autoClaimer.Start(ctx)
	if err != nil {
		return nil, err
	}

	return autoClaimer, nil
}
