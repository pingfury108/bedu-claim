package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Subject 表示教育系统中的学科
type Subject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Filter 表示API响应中使用的过滤器
type Filter struct {
	ID   string    `json:"id"`
	Name string    `json:"name"`
	Type string    `json:"type"`
	List []Subject `json:"list"`
}

// LabelResponse 表示来自getLabel API的响应
type LabelResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		Filter []Filter `json:"filter"`
	} `json:"data"`
}

// TaskItem 表示任务列表响应中的单个任务项
type TaskItem struct {
	TaskID       int    `json:"taskID"`
	ClueID       int    `json:"clueID"`
	Brief        string `json:"brief"`
	Step         int    `json:"step"`
	Subject      int    `json:"subject"`
	State        int    `json:"state"`
	StepName     string `json:"stepName"`
	SubjectName  string `json:"subjectName"`
	ClueType     int    `json:"clueType"`
	ClueTypeName string `json:"clueTypeName"`
	StateName    string `json:"stateName"`
	CreateTime   string `json:"createTime"`
	DispatchTime string `json:"dispatchTime"`
}

// TaskListData 表示任务列表响应中的数据字段
type TaskListData struct {
	Total int        `json:"total"`
	List  []TaskItem `json:"list"`
}

// TaskListResponse 表示来自任务列表API的响应
type TaskListResponse struct {
	Errno  int          `json:"errno"`
	Errmsg string       `json:"errmsg"`
	Data   TaskListData `json:"data"`
}

// ClaimResponse 表示来自认领API的响应
type ClaimResponse struct {
	Errno  int         `json:"errno"`
	Errmsg string      `json:"errmsg"`
	Data   interface{} `json:"data"`
}

// GetAuditTaskLabel 从服务器获取审核任务标签
func GetAuditTaskLabel(taskType, serverBaseURL, cookie string) (*LabelResponse, error) {
	if taskType == "" {
		taskType = "audittask"
	}

	// 使用传入的服务器基础URL和cookie
	apiURL := fmt.Sprintf("%s/edushop/question/%s/getlabel", serverBaseURL, taskType)

	// 创建请求
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// 如果cookie存在，设置头部
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	// 设置User-Agent头部，模拟正常浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error! status: %d, URL: %s", resp.StatusCode, apiURL)
	}

	// 读取并解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData LabelResponse
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, err
	}

	// 如果响应成功，添加额外的学科
	if responseData.Errno == 0 && len(responseData.Data.Filter) > 0 {
		var subjectFilter *Filter
		for i := range responseData.Data.Filter {
			if responseData.Data.Filter[i].ID == "subject" {
				subjectFilter = &responseData.Data.Filter[i]
				break
			}
		}

		if subjectFilter != nil {
			// 要添加的额外学科
			additionalSubjects := []Subject{}

			// 如果学科不存在，则添加
			for _, subject := range additionalSubjects {
				exists := false
				for _, existingSubject := range subjectFilter.List {
					if existingSubject.ID == subject.ID {
						exists = true
						break
					}
				}

				if !exists {
					subjectFilter.List = append(subjectFilter.List, subject)
				}
			}
		}
	}

	return &responseData, nil
}

// GetAuditTaskList 从服务器获取审核任务列表
func GetAuditTaskList(options map[string]interface{}, serverBaseURL, cookie string) (*TaskListResponse, error) {
	// Set default values
	pn := 1
	rn := 20
	clueID := ""
	clueType := 1
	step := 1
	subject := 2
	taskType := "audittask"

	// Override with provided options
	if val, ok := options["pn"].(int); ok {
		pn = val
	}
	if val, ok := options["rn"].(int); ok {
		rn = val
	}
	if val, ok := options["clueID"].(string); ok {
		clueID = val
	}
	if val, ok := options["clueType"].(int); ok {
		clueType = val
	}
	if val, ok := options["step"].(int); ok {
		step = val
	}
	if val, ok := options["subject"].(int); ok {
		subject = val
	}
	if val, ok := options["taskType"].(string); ok && val != "" {
		taskType = val
	}

	// 创建查询参数
	queryParams := url.Values{}
	queryParams.Add("pn", strconv.Itoa(pn))
	queryParams.Add("rn", strconv.Itoa(rn))
	queryParams.Add("clueID", clueID)
	queryParams.Add("clueType", strconv.Itoa(clueType))
	queryParams.Add("step", strconv.Itoa(step))
	queryParams.Add("subject", strconv.Itoa(subject))

	// 使用传入的服务器基础URL和cookie
	apiURL := fmt.Sprintf("%s/edushop/question/%s/list?%s", serverBaseURL, taskType, queryParams.Encode())

	// 创建请求
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// 如果cookie存在，设置头部
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	// 设置User-Agent头部，模拟正常浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error! status: %d, URL: %s", resp.StatusCode, apiURL)
	}

	// 读取并解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData TaskListResponse
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, err
	}

	return &responseData, nil
}

// UserInfoResponse 表示用户信息的响应
type UserInfoResponse struct {
	Errno  int    `json:"errno"`
	Errmsg string `json:"errmsg"`
	Data   struct {
		RoleLinks []string `json:"roleLinks"`
		RoleNames []string `json:"roleNames"`
		UserName  string   `json:"userName"`
		Avatar    string   `json:"avatar"`
	} `json:"data"`
}

// GetUserInfo 获取用户信息
func GetUserInfo(serverBaseURL, cookie string) (*UserInfoResponse, error) {
	// 构建用户信息API URL
	apiURL := fmt.Sprintf("%s/edushop/user/common/info", serverBaseURL)

	// 创建请求
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// 如果cookie存在，设置头部
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	// 设置User-Agent头部，模拟正常浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error! status: %d, URL: %s", resp.StatusCode, apiURL)
	}

	// 读取并解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData UserInfoResponse
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, err
	}

	return &responseData, nil
}

// ClaimAuditTask 认领一个或多个审核任务
func ClaimAuditTask(taskIDs []string, taskType, serverBaseURL, cookie string) (*ClaimResponse, error) {
	if taskType == "" {
		taskType = "audittask"
	}

	commitType := "audittaskcommit"
	if taskType == "producetask" {
		commitType = "producetaskcommit"
	}

	// 准备请求体
	var requestBody map[string]interface{}
	if taskType == "producetask" {
		// 将字符串ID转换为整数ID
		clueIDs := make([]uint64, 0, len(taskIDs))
		for _, idStr := range taskIDs {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("无法将clueID '%s'转换为整数: %v", idStr, err)
			}
			clueIDs = append(clueIDs, id)
		}
		requestBody = map[string]interface{}{
			"clueIDs": clueIDs,
		}
	} else {
		// 将字符串ID转换为整数ID，对于audittask也需要转换
		numericTaskIDs := make([]uint64, 0, len(taskIDs))
		for _, idStr := range taskIDs {
			id, err := strconv.ParseUint(idStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("无法将taskID '%s'转换为整数: %v", idStr, err)
			}
			numericTaskIDs = append(numericTaskIDs, id)
		}
		requestBody = map[string]interface{}{
			"taskIDs": numericTaskIDs,
		}
	}

	// 将请求体转换为JSON
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	// 使用传入的服务器基础URL和cookie
	apiURL := fmt.Sprintf("%s/edushop/question/%s/claim", serverBaseURL, commitType)

	// 创建请求
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	// 设置User-Agent头部，模拟正常浏览器
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error! status: %d, URL: %s", resp.StatusCode, apiURL)
	}

	// 读取并解析响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseData ClaimResponse
	err = json.Unmarshal(body, &responseData)
	if err != nil {
		return nil, err
	}

	return &responseData, nil
}
