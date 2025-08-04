import React, { useState, useEffect, useCallback, useRef } from 'react';
import { StartAutoClaiming, StopAutoClaiming, GetAutoClaimStatus, GetTaskLabels } from '../wailsjs/go/main/App.js';
import { main } from '../wailsjs/go/models.js';

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

  // 处理任务类型改变
  const handleTaskTypeChange = useCallback(async (taskType: string) => {
    setSelectedTaskType(taskType);
    setIsLoading(true);
    try {
      const response = await GetTaskLabels(taskType, cookie);
      if (response && response.errno === 0) {
        setFilterData(response.data.filter || []);
        // 设置默认选择
        const stepFilter = response.data.filter?.find((f: Filter) => f.id === 'step');
        const subjectFilter = response.data.filter?.find((f: Filter) => f.id === 'subject');
        const clueTypeFilter = response.data.filter?.find((f: Filter) => f.id === 'clueType');
        
        if (stepFilter?.list && stepFilter.list.length > 0) setSelectedGrade(stepFilter.list[0].name);
        if (subjectFilter?.list && subjectFilter.list.length > 0) setSelectedSubject(subjectFilter.list[0].name);
        if (clueTypeFilter?.list && clueTypeFilter.list.length > 0) setSelectedType(clueTypeFilter.list[0].name);
      }
    } catch (error) {
      console.error('获取标签数据失败:', error);
      setUserInfoError('获取筛选数据失败，请检查网络连接');
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
        // ServerBaseURL 已在Go代码中硬编码为 DefaultServerURL
        Cookie: cookie,
        TaskType: selectedTaskType,
        ClaimLimit: claimLimit,
        Interval: refreshInterval,
        MaxPages: 0,
        ConcurrentClaims: 10,
        StepID: stepItem?.id || 0,
        SubjectID: subjectItem?.id || 0,
        ClueTypeID: clueTypeItem?.id || 0,
        IncludeKeywords: includeKeywords,
        ExcludeKeywords: excludeKeywords,
        StartTime: startTime ? startTime.replace('T', ' ') + ':00' : '',
        EndTime: endTime ? endTime.replace('T', ' ') + ':00' : '',
      };

      const response = await StartAutoClaiming(config);
      
      if (response.success) {
        setAutoClaimingActive(true);
        // 开始定期检查状态
        statusIntervalRef.current = setInterval(checkAutoClaimStatus, 2000);
      } else {
        setUserInfoError(response.message);
      }
    } catch (error) {
      setUserInfoError(`启动失败: ${(error as Error).message}`);
    } finally {
      setIsClaimingButtonLoading(false);
    }
  }, [cookie, selectedTaskType, claimLimit, refreshInterval, filterData, selectedGrade, selectedSubject, selectedType, includeKeywords, excludeKeywords, startTime, endTime]);

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

  // 组件初始化
  useEffect(() => {
    // 从localStorage加载设置
    const savedCookie = localStorage.getItem('serverCookie') || '';
    const savedStartTime = localStorage.getItem('clueStartTime') || '';
    const savedEndTime = localStorage.getItem('clueEndTime') || '';
    
    setCookie(savedCookie);
    setStartTime(savedStartTime);
    setEndTime(savedEndTime);

    // 清理函数
    return () => {
      if (statusIntervalRef.current) {
        clearInterval(statusIntervalRef.current);
      }
    };
  }, []);

  // 当cookie配置完成后加载标签数据
  useEffect(() => {
    if (cookie) {
      handleTaskTypeChange(selectedTaskType);
    }
  }, [cookie, handleTaskTypeChange]);

  return (
    <div className="w-full mt-2">

      {isLoading && (
        <div className="flex items-center justify-center my-4">
          <span className="loading loading-spinner loading-sm mr-2"></span>
          <span className="text-sm">加载筛选数据中...</span>
        </div>
      )}

      {/* 筛选配置区域 */}
      <div className="flex flex-col gap-4 mb-4">
        <input
          type="text"
          value={cookie}
          onChange={(e) => {
            setCookie(e.target.value);
            localStorage.setItem('serverCookie', e.target.value);
          }}
          className="input input-bordered input-sm w-full"
          placeholder="Cookie"
        />
        
        <div className="form-control">
          <label className="label py-1">
            <span className="label-text text-sm font-medium">任务类型</span>
          </label>
          <select
            className="select select-bordered select-sm w-full"
            value={selectedTaskType}
            onChange={(e) => handleTaskTypeChange(e.target.value)}
          >
            <option value="audittask">审核任务</option>
            <option value="producetask">生产任务</option>
          </select>
        </div>

        {filterData.length > 0 && (
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
              className="input input-bordered input-sm w-32 text-center"
            />
            <span className="text-sm">个</span>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <span className="text-sm font-medium">轮询间隔：</span>
          <div className="flex items-center gap-2">
            <input
              type="number"
              min="0.1"
              max="60"
              step="0.1"
              value={refreshInterval}
              onChange={(e) => setRefreshInterval(Number(e.target.value))}
              className="input input-bordered input-sm w-32 text-center"
            />
            <span className="text-sm">秒</span>
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
        {userInfoError && (
          <div className="alert alert-error mb-2 p-2 text-sm">
            <svg xmlns="http://www.w3.org/2000/svg" className="stroke-current shrink-0 h-4 w-4" fill="none" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{userInfoError}</span>
          </div>
        )}

        {/* 显示当前设置概述 */}
        {!autoClaimingActive && (
          <div className="mb-2 p-2 bg-base-200 rounded text-xs">
            <div className="font-medium mb-1">📋 当前设置:</div>
            <div>任务类型: {selectedTaskType === 'producetask' ? '生产' : '审核'} | 上限: {claimLimit}个 | 间隔: {refreshInterval}秒</div>
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
        {claimStatus && (
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
            disabled={isClaimingButtonLoading}
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