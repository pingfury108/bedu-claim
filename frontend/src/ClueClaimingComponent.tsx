import React, { useState, useEffect, useCallback, useRef } from 'react';
import { StartAutoClaiming, StopAutoClaiming, GetAutoClaimStatus, GetTaskLabels, GetUserInfo } from '../wailsjs/go/main/App.js';
import { main } from '../wailsjs/go/models.js';
import { BrowserOpenURL } from '../wailsjs/runtime/runtime.js';

// 类型定义
type Filter = {
  id: string;
  name: string;
  type: string;
  list: { id: number; name: string }[];
};

type AutoClaimStatusType = {
  success: boolean;
  message: string;
  isActive: boolean;
  successfulClaims: number;
  lastError: string;
};

export default function ClueClaimingComponent() {
  // 状态变量
  const [selectedTaskType, setSelectedTaskType] = useState('audittask');
  const [selectedGrade, setSelectedGrade] = useState('');
  const [selectedSubject, setSelectedSubject] = useState('');
  const [selectedType, setSelectedType] = useState('');
  const [claimLimit, setClaimLimit] = useState(10);
  const [refreshInterval, setRefreshInterval] = useState(1.0);
  const [timeUnit, setTimeUnit] = useState<'seconds' | 'milliseconds'>('seconds');
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');
  const [includeKeywords, setIncludeKeywords] = useState<string[]>([]);
  const [excludeKeywords, setExcludeKeywords] = useState<string[]>([]);
  const [newIncludeKeyword, setNewIncludeKeyword] = useState('');
  const [newExcludeKeyword, setNewExcludeKeyword] = useState('');
  const [filterData, setFilterData] = useState<Filter[]>([]);
  const [autoClaimingActive, setAutoClaimingActive] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);
  const [isClaimingButtonLoading, setIsClaimingButtonLoading] = useState<boolean>(false);
  const [userInfoError, setUserInfoError] = useState<string>('');
  const [cookie, setCookie] = useState<string>('');
  const [claimStatus, setClaimStatus] = useState<AutoClaimStatusType | null>(null);
  const [userInfo, setUserInfo] = useState<{ username: string; avatar: string } | null>(null);
  const [userInfoLoading, setUserInfoLoading] = useState(false);
  const [showAuthModal, setShowAuthModal] = useState(false);
  const [authType, setAuthType] = useState<'official' | 'custom'>('official');
  const [authUsername, setAuthUsername] = useState('');

  const isUserInteractionRef = useRef(false);
  const statusIntervalRef = useRef<number | null>(null);

  // 获取今天开始和结束时间的工具函数
  const getTodayStartTime = () => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    return today.toISOString().slice(0, 16);
  };

  const getTodayEndTime = () => {
    const today = new Date();
    today.setHours(23, 59, 59, 999);
    return today.toISOString().slice(0, 16);
  };

  // 显示toast通知
  const showToast = (message: string, type: 'error' | 'success' | 'warning' | 'info' = 'error') => {
    const toast = document.createElement('div');
    toast.className = 'toast toast-end z-50';

    const alertClass = type === 'error' ? 'alert-error' :
                      type === 'success' ? 'alert-success' :
                      type === 'warning' ? 'alert-warning' : 'alert-info';

    const iconPath = type === 'error' ? 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z' :
                     type === 'success' ? 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z' :
                     'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z';

    toast.innerHTML = `
      <div class="alert ${alertClass}">
        <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="${iconPath}" />
        </svg>
        <span>${message}</span>
      </div>
    `;

    document.body.appendChild(toast);
    setTimeout(() => {
      if (document.body.contains(toast)) {
        document.body.removeChild(toast);
      }
    }, 4000);
  };

  // 处理任务类型改变
  const handleTaskTypeChange = useCallback(async (taskType: string) => {
    setSelectedTaskType(taskType);
    setIsLoading(true);
    // 清除旧标签数据和选择
    setFilterData([]);
    setSelectedGrade('');
    setSelectedSubject('');
    setSelectedType('');

    try {
      const response = await GetTaskLabels(taskType, cookie);
      console.log('GetTaskLabels response:', response);

      if (response && response.errno === 0) {
        setFilterData(response.data.filter || []);
        // 设置默认选择
        const stepFilter = response.data.filter?.find((f: Filter) => f.id === 'step');
        const subjectFilter = response.data.filter?.find((f: Filter) => f.id === 'subject');
        const clueTypeFilter = response.data.filter?.find((f: Filter) => f.id === 'clueType');

        if (stepFilter?.list && stepFilter.list.length > 0) setSelectedGrade(stepFilter.list[0].name);
        if (subjectFilter?.list && subjectFilter.list.length > 0) setSelectedSubject(subjectFilter.list[0].name);
        if (clueTypeFilter?.list && clueTypeFilter.list.length > 0) setSelectedType(clueTypeFilter.list[0].name);
      } else {
        const errorMsg = response?.errmsg || response?.message || response?.msg || response?.error || `错误码: ${response?.errno || '未知'}`;
        showToast(`获取标签信息失败: ${errorMsg}`, 'error');
      }
    } catch (error) {
      console.error('获取标签数据失败:', error);
      const errorMessage = error instanceof Error ? error.message : '网络连接失败';
      showToast(`获取标签信息失败: ${errorMessage}`, 'error');
    } finally {
      setIsLoading(false);
    }
  }, [cookie]);

  // 启动自动认领
  const startAutoClaiming = useCallback(async () => {
    setIsClaimingButtonLoading(true);
    setUserInfoError('');

    try {
      // 获取选择的筛选器ID
      const stepFilter = filterData.find(f => f.id === 'step');
      const subjectFilter = filterData.find(f => f.id === 'subject');
      const clueTypeFilter = filterData.find(f => f.id === 'clueType');

      const stepItem = stepFilter?.list.find(item => item.name === selectedGrade);
      const subjectItem = subjectFilter?.list.find(item => item.name === selectedSubject);
      const clueTypeItem = clueTypeFilter?.list.find(item => item.name === selectedType);

  
      const config: main.AutoClaimConfig = {
        ServerBaseURL: '', // 已在Go代码中硬编码为 DefaultServerURL
        Cookie: cookie,
        TaskType: selectedTaskType,
        ClaimLimit: claimLimit,
        Interval: timeUnit === 'seconds' ? refreshInterval : refreshInterval / 1000,
        MaxPages: 0,
        ConcurrentClaims: 10,
        StepID: stepItem?.id || 0,
        SubjectID: subjectItem?.id || 0,
        ClueTypeID: clueTypeItem?.id || 0,
        IncludeKeywords: includeKeywords,
        ExcludeKeywords: excludeKeywords,
        StartTime: startTime ? startTime.replace('T', ' ') + ':00' : '',
        EndTime: endTime ? endTime.replace('T', ' ') + ':00' : '',
        authType: authType,
        authUsername: authUsername,
      };

      const response = await StartAutoClaiming(config);

      if (response.success) {
        setAutoClaimingActive(true);
        // 开始定期检查状态
        statusIntervalRef.current = setInterval(checkAutoClaimStatus, 2000);
      } else {
        showToast(`启动失败: ${response.message}`, 'error');
      }
    } catch (error) {
      showToast(`启动失败: ${(error as Error).message}`, 'error');
    } finally {
      setIsClaimingButtonLoading(false);
    }
  }, [cookie, selectedTaskType, claimLimit, refreshInterval, filterData, selectedGrade, selectedSubject, selectedType, includeKeywords, excludeKeywords, startTime, endTime, authType, authUsername]);

  // 停止自动认领
  const stopAutoClaiming = useCallback(async () => {
    try {
      const response = await StopAutoClaiming();
      if (response.success) {
        setAutoClaimingActive(false);
        if (statusIntervalRef.current) {
          clearInterval(statusIntervalRef.current);
          statusIntervalRef.current = null;
        }
      }
    } catch (error) {
      console.error('停止自动认领失败:', error);
    }
  }, []);

  // 检查自动认领状态
  const checkAutoClaimStatus = useCallback(async () => {
    try {
      const response = await GetAutoClaimStatus();
      if (response.success) {
        setClaimStatus(response);
        if (!response.isActive && autoClaimingActive) {
          // 任务已完成或停止
          setAutoClaimingActive(false);
          if (statusIntervalRef.current) {
            clearInterval(statusIntervalRef.current);
            statusIntervalRef.current = null;
          }
        }
      }
    } catch (error) {
      console.error('获取状态失败:', error);
    }
  }, [autoClaimingActive]);

  // 获取用户信息
  const fetchUserInfo = useCallback(async (cookieValue: string) => {
    if (!cookieValue.trim()) {
      setUserInfo(null);
      return;
    }

    setUserInfoLoading(true);
    try {
      const response = await GetUserInfo(cookieValue);
      if (response && response.errno === 0) {
        setUserInfo({
          username: response.data.userName || '未知用户',
          avatar: response.data.avatar || ''
        });
      } else {
        setUserInfo(null);
      }
    } catch (error) {
      console.error('获取用户信息失败:', error);
      setUserInfo(null);
    } finally {
      setUserInfoLoading(false);
    }
  }, []);

  // 组件初始化
  useEffect(() => {
    // 从localStorage加载设置
    const savedCookie = localStorage.getItem('serverCookie') || '';
    const savedStartTime = localStorage.getItem('clueStartTime') || '';
    const savedEndTime = localStorage.getItem('clueEndTime') || '';
    const savedAuthType = localStorage.getItem('authType') as 'official' | 'custom' || 'official';
    const savedAuthUsername = localStorage.getItem('authUsername') || '';

    setCookie(savedCookie);
    setStartTime(savedStartTime);
    setEndTime(savedEndTime);
    setAuthType(savedAuthType);
    setAuthUsername(savedAuthUsername);

    // 如果已有cookie，获取用户信息
    if (savedCookie) {
      fetchUserInfo(savedCookie);
    }

    // 清理函数
    return () => {
      if (statusIntervalRef.current) {
        clearInterval(statusIntervalRef.current);
      }
    };
  }, [fetchUserInfo]);

  
  // 当cookie或任务类型变化时加载标签数据
  useEffect(() => {
    if (cookie) {
      handleTaskTypeChange(selectedTaskType);
    }
  }, [cookie, selectedTaskType, handleTaskTypeChange]);

  return (
    <div className="w-full mt-2">
      {/* 授权设置弹窗 */}
      {showAuthModal && (
        <div className="modal modal-open">
          <div className="modal-box max-w-md">
            <div className="flex justify-between items-center mb-4">
              <h3 className="font-bold text-lg">软件授权设置</h3>
              <button
                className="btn btn-sm btn-circle btn-ghost"
                onClick={() => setShowAuthModal(false)}
              >
                ✕
              </button>
            </div>
            
            {/* 授权类型选择 */}
            <div className="form-control mb-4">
              <label className="label">
                <span className="label-text font-medium">授权类型</span>
              </label>
              <div className="join w-full">
                <button
                  className={`btn join-item flex-1 ${authType === 'official' ? 'btn-primary' : 'btn-outline'}`}
                  onClick={() => setAuthType('official')}
                >
                  官方授权
                </button>
                <button
                  className={`btn join-item flex-1 ${authType === 'custom' ? 'btn-primary' : 'btn-outline'}`}
                  onClick={() => setAuthType('custom')}
                >
                  定制授权
                </button>
              </div>
            </div>

            {/* 根据授权类型显示不同内容 */}
            {authType === 'official' ? (
              <div className="form-control">
                <label className="label">
                  <span className="label-text">用户名</span>
                </label>
                <input
                  type="text"
                  value={authUsername}
                  onChange={(e) => setAuthUsername(e.target.value)}
                  className="input input-bordered w-full"
                  placeholder="请输入官方授权用户名"
                />
                <label className="label">
                  <span className="label-text-alt text-info">
                    请输入您的官方授权用户名进行验证
                  </span>
                </label>
              </div>
            ) : (
              <div className="form-control">
                <label className="label">
                  <span className="label-text">当前用户</span>
                </label>
                <div className="input input-bordered w-full bg-base-200">
                  {userInfo ? userInfo.username : '请先输入百度教育Cookie'}
                </div>
                <label className="label">
                  <span className="label-text-alt text-info">
                    使用百度教育Cookie关联的用户身份进行授权
                  </span>
                </label>
              </div>
            )}

            {/* 弹窗底部按钮 */}
            <div className="modal-action">
              <button
                className="btn btn-primary"
                onClick={() => {
                  // 保存授权设置到localStorage
                  localStorage.setItem('authType', authType);
                  localStorage.setItem('authUsername', authUsername);
                  setShowAuthModal(false);
                  showToast('授权设置已保存', 'success');
                }}
              >
                保存设置
              </button>
              <button
                className="btn btn-ghost"
                onClick={() => setShowAuthModal(false)}
              >
                取消
              </button>
            </div>
          </div>
          <div className="modal-backdrop">
            <button onClick={() => setShowAuthModal(false)}>close</button>
          </div>
        </div>
      )}


      {/* 筛选配置区域 */}
      <div className="flex flex-col gap-4 mb-4">
        <div className="form-control">
          <div className="flex justify-between items-center mb-2">
            <span className="label-text text-sm font-medium">软件授权</span>
            <button
              className="btn btn-outline btn-xs btn-primary"
              onClick={() => {
                    setShowAuthModal(true);
                }}
            >
              设置授权
            </button>
          </div>
          <label className="label py-1 flex justify-between items-center">
            <span className="label-text text-sm font-medium">百度教育 Cookie</span>
            <div className="flex items-center gap-2">
              {userInfo && (
                <span className="text-xs text-base-content/70">
                  👤 {userInfo.username}
                </span>
              )}
              {userInfoLoading && (
                <span className="loading loading-spinner loading-xs"></span>
              )}
              <button
                type="button"
                className="text-info hover:text-primary btn btn-ghost btn-xs p-0 min-h-0 h-auto"
                onClick={(e) => {
                  e.stopPropagation();
                  BrowserOpenURL('https://www.bilibili.com/video/BV11YVNzGESS/');
                }}
                title="查看视频教程"
                tabIndex={-1}
              >
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor" className="w-4 h-4">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z" />
                </svg>
              </button>
            </div>
          </label>
          <input
            type="text"
            value={cookie}
            onChange={(e) => {
              const newCookie = e.target.value;
              setCookie(newCookie);
              localStorage.setItem('serverCookie', newCookie);
              fetchUserInfo(newCookie);
            }}
            className="input input-bordered input-sm w-full"
            placeholder="请输入Cookie"
          />
        </div>

        <div className="form-control">
          <label className="label py-1">
            <span className="label-text text-sm font-medium">任务类型</span>
          </label>
          <select
            className="select select-bordered select-sm w-full"
            value={selectedTaskType}
            onChange={(e) => handleTaskTypeChange(e.target.value)}
          >
            <option value="audittask">审核</option>
            <option value="producetask">生产</option>
          </select>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-2">
            <span className="loading loading-spinner loading-sm mr-2"></span>
            <span className="text-sm">加载筛选数据中...</span>
          </div>
        ) : filterData.length > 0 && (
          <div className="grid grid-cols-3 gap-2">
            {/* 学段选择 */}
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text text-xs">学段</span>
              </label>
              <select
                className="select select-bordered select-sm w-full"
                value={selectedGrade}
                onChange={(e) => {
                  isUserInteractionRef.current = true;
                  setSelectedGrade(e.target.value);
                }}
              >
                {filterData.find(f => f.id === 'step')?.list.map(item => (
                  <option key={item.id} value={item.name}>{item.name}</option>
                ))}
              </select>
            </div>

            {/* 学科选择 */}
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text text-xs">学科</span>
              </label>
              <select
                className="select select-bordered select-sm w-full"
                value={selectedSubject}
                onChange={(e) => {
                  isUserInteractionRef.current = true;
                  setSelectedSubject(e.target.value);
                }}
              >
                {filterData.find(f => f.id === 'subject')?.list.map(item => (
                  <option key={item.id} value={item.name}>{item.name}</option>
                ))}
              </select>
            </div>

            {/* 类型选择 */}
            <div className="form-control">
              <label className="label py-1">
                <span className="label-text text-xs">类型</span>
              </label>
              <select
                className="select select-bordered select-sm w-full"
                value={selectedType}
                onChange={(e) => {
                  isUserInteractionRef.current = true;
                  setSelectedType(e.target.value);
                }}
              >
                {filterData.find(f => f.id === 'clueType')?.list.map(item => (
                  <option key={item.id} value={item.name}>{item.name}</option>
                ))}
              </select>
            </div>
          </div>
        )}
      </div>

      <div className="divider text-sm my-2">⚙️ 自动认领设置</div>

      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">认领上限：</span>
          <div className="flex items-center gap-2">
            <input
              type="number"
              min="1"
              max="1000"
              value={claimLimit}
              onChange={(e) => setClaimLimit(Number(e.target.value))}
              className="input input-bordered input-sm w-24 text-center"
            />
            <span className="text-sm w-18 text-center">个</span>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">轮询间隔：</span>
          <div className="flex items-center gap-2">
            <input
              type="number"
              min={timeUnit === 'seconds' ? "0.1" : "10"}
              max={timeUnit === 'seconds' ? "60" : "60000"}
              step={timeUnit === 'seconds' ? "0.1" : "10"}
              value={refreshInterval}
              onChange={(e) => setRefreshInterval(Number(e.target.value))}
              className="input input-bordered input-sm w-24 text-center"
            />
            <select
              value={timeUnit}
              onChange={(e) => {
                const newUnit = e.target.value as 'seconds' | 'milliseconds';
                setTimeUnit(newUnit);
                // Convert value when switching units
                if (newUnit === 'milliseconds' && timeUnit === 'seconds') {
                  setRefreshInterval(refreshInterval * 1000);
                } else if (newUnit === 'seconds' && timeUnit === 'milliseconds') {
                  setRefreshInterval(refreshInterval / 1000);
                }
              }}
              className="select select-bordered select-sm w-18"
            >
              <option value="seconds">秒</option>
              <option value="milliseconds">毫秒</option>
            </select>
          </div>
        </div>
      </div>

      {/* 只在生产任务时显示时间过滤 */}
      {selectedTaskType === 'producetask' && (
        <>
          <div className="divider text-sm my-2">📅 发布时间过滤</div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-2">
            <div>
              <input
                type="datetime-local"
                value={startTime}
                onChange={(e) => {
                  setStartTime(e.target.value);
                  localStorage.setItem('clueStartTime', e.target.value);
                }}
                className="input input-bordered input-sm w-full"
                placeholder="开始时间"
              />
            </div>

            <div>
              <input
                type="datetime-local"
                value={endTime}
                onChange={(e) => {
                  setEndTime(e.target.value);
                  localStorage.setItem('clueEndTime', e.target.value);
                }}
                className="input input-bordered input-sm w-full"
                placeholder="结束时间"
              />
            </div>

            <div>
              <button
                className="btn btn-outline btn-sm w-full"
                onClick={() => {
                  const todayStart = getTodayStartTime();
                  const todayEnd = getTodayEndTime();
                  setStartTime(todayStart);
                  setEndTime(todayEnd);
                  localStorage.setItem('clueStartTime', todayStart);
                  localStorage.setItem('clueEndTime', todayEnd);
                }}
              >
                重置为今天
              </button>
            </div>
          </div>
        </>
      )}

      <div className="divider text-sm my-2">🔍 关键词过滤</div>

      <div className="flex flex-col gap-2">
        <div className="form-control">
          <label className="label py-1">
            <span className="label-text text-sm font-medium">包含关键词</span>
          </label>
          <div className="flex gap-1 flex-wrap mb-1">
            {includeKeywords.map((keyword, index) => (
              <div key={index} className="badge badge-primary gap-1">
                {keyword}
                <button
                  className="btn btn-ghost btn-xs p-0"
                  onClick={() => setIncludeKeywords(includeKeywords.filter((_, i) => i !== index))}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
          </div>
          <div className="join w-full">
            <input
              type="text"
              value={newIncludeKeyword}
              onChange={(e) => setNewIncludeKeyword(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter' && newIncludeKeyword.trim()) {
                  setIncludeKeywords([...includeKeywords, newIncludeKeyword.trim()]);
                  setNewIncludeKeyword('');
                }
              }}
              className="input input-sm input-bordered join-item w-full"
              placeholder="输入关键词，按回车添加"
            />
            <button
              className="btn btn-sm join-item"
              onClick={() => {
                if (newIncludeKeyword.trim()) {
                  setIncludeKeywords([...includeKeywords, newIncludeKeyword.trim()]);
                  setNewIncludeKeyword('');
                }
              }}
            >
              添加
            </button>
          </div>
        </div>

        <div className="form-control">
          <label className="label py-1">
            <span className="label-text text-sm font-medium">排除关键词</span>
          </label>
          <div className="flex gap-1 flex-wrap mb-1">
            {excludeKeywords.map((keyword, index) => (
              <div key={index} className="badge badge-secondary gap-1">
                {keyword}
                <button
                  className="btn btn-ghost btn-xs p-0"
                  onClick={() => setExcludeKeywords(excludeKeywords.filter((_, i) => i !== index))}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            ))}
          </div>
          <div className="join w-full">
            <input
              type="text"
              value={newExcludeKeyword}
              onChange={(e) => setNewExcludeKeyword(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter' && newExcludeKeyword.trim()) {
                  setExcludeKeywords([...excludeKeywords, newExcludeKeyword.trim()]);
                  setNewExcludeKeyword('');
                }
              }}
              className="input input-sm input-bordered join-item w-full"
              placeholder="输入关键词，按回车添加"
            />
            <button
              className="btn btn-sm join-item"
              onClick={() => {
                if (newExcludeKeyword.trim()) {
                  setExcludeKeywords([...excludeKeywords, newExcludeKeyword.trim()]);
                  setNewExcludeKeyword('');
                }
              }}
            >
              添加
            </button>
          </div>
        </div>
      </div>

      <div className="mt-4">

        {/* 显示当前设置概述 */}
        {!autoClaimingActive && (
          <div className="mb-2 p-2 bg-base-200 rounded text-xs">
            <div className="font-medium mb-1">📋 当前设置:</div>
            <div>任务类型: {selectedTaskType === 'producetask' ? '生产' : '审核'} | 上限: {claimLimit}个 | 间隔: {refreshInterval}{timeUnit === 'seconds' ? '秒' : '毫秒'}</div>
            {(includeKeywords.length > 0 || excludeKeywords.length > 0) && (
              <div>
                {includeKeywords.length > 0 && `包含: ${includeKeywords.join(', ')} `}
                {excludeKeywords.length > 0 && `排除: ${excludeKeywords.join(', ')}`}
              </div>
            )}
            {selectedTaskType === 'producetask' && (startTime || endTime) && (
              <div>
                时间过滤: {startTime ? `从 ${startTime.replace('T', ' ')}` : '无开始时间'} {endTime ? `到 ${endTime.replace('T', ' ')}` : '无结束时间'}
              </div>
            )}
          </div>
        )}

        {/* 状态显示区域 */}
        {autoClaimingActive && claimStatus && (
          <div className="mb-2 p-3 bg-base-100 rounded-lg shadow-sm">
            <div className="flex justify-between items-center">
              <span className="font-medium text-sm">📊 认领状态:</span>
              <span className={`badge ${claimStatus.isActive ? 'badge-success' : 'badge-neutral'}`}>
                {claimStatus.isActive ? '运行中' : '已停止'}
              </span>
            </div>
            <div className="mt-2 text-sm">
              成功认领: <span className="font-mono font-bold text-success">{claimStatus.successfulClaims}</span> 个任务
            </div>
            {claimStatus.lastError && (
              <div className="text-error text-xs mt-1 bg-error/10 p-2 rounded">
                ❌ {claimStatus.lastError}
              </div>
            )}
          </div>
        )}

        {/* 操作按钮 */}
        {autoClaimingActive ? (
          <button
            className="btn btn-error btn-sm w-full"
            onClick={stopAutoClaiming}
          >
            <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
            停止自动认领
          </button>
        ) : (
          <button
            className="btn btn-primary btn-sm w-full"
            onClick={() => startAutoClaiming()}
            disabled={isClaimingButtonLoading || isLoading || filterData.length === 0}
          >
            {isClaimingButtonLoading ? (
              <>
                <span className="loading loading-spinner loading-xs mr-2"></span>
                <span>启动中...</span>
              </>
            ) : (
              <>
                <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                🚀 启动自动认领
              </>
            )}
          </button>
        )}
      </div>
    </div>
  );
}
