// lib/hooks/api.ts - React hooks for API calls
'use client';

import { useState, useEffect, useCallback } from 'react';
import { apiClient, ActivityEvent, Project, StorageStats } from '@/lib/api-client';

// Generic hook for API calls
export function useAPI<T>(
  apiCall: () => Promise<T>,
  dependencies: any[] = [],
  initialData?: T
) {
  const [data, setData] = useState<T | undefined>(initialData);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await apiCall();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      console.error('API Error:', err);
    } finally {
      setLoading(false);
    }
  }, dependencies);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const refetch = useCallback(() => {
    return fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch };
}

// Specific hooks for different data types

export function useProjects() {
  return useAPI(
    () => apiClient.getProjects(),
    [],
    { success: true, projects: [] as Project[] }
  );
}

export function useStorageStats() {
  return useAPI(() => apiClient.getStorageStats());
}

export function useProjectLocks(projectId: string) {
  return useAPI(
    () => apiClient.getProjectLocks(projectId),
    [projectId],
    { success: true, locks: [] }
  );
}

export function useProjectPresence(projectId: string) {
  return useAPI(
    () => apiClient.getProjectPresence(projectId),
    [projectId],
    { success: true, presence: [] }
  );
}

export function useActivityFeed(projectId: string, limit: number = 50) {
  return useAPI(
    () => apiClient.getActivityFeed(projectId, limit),
    [projectId, limit],
    { success: true, activities: [] }
  );
}

// Real-time hooks with auto-refresh
export function useRealtimePresence(projectId: string, refreshInterval: number = 5000) {
  const { data, loading, error, refetch } = useProjectPresence(projectId);

  useEffect(() => {
    if (!projectId) return;

    const interval = setInterval(() => {
      refetch();
    }, refreshInterval);

    return () => clearInterval(interval);
  }, [projectId, refreshInterval, refetch]);

  return { data, loading, error, refetch };
}

export function useRealtimeActivity(projectId: string, refreshInterval: number = 10000) {
  const { data, loading, error, refetch } = useActivityFeed(projectId, 20);

  useEffect(() => {
    if (!projectId) return;

    const interval = setInterval(() => {
      refetch();
    }, refreshInterval);

    return () => clearInterval(interval);
  }, [projectId, refreshInterval, refetch]);

  return { data, loading, error, refetch };
}

// Mutation hooks for actions
export function useFileLock() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const lockFile = useCallback(async (
    projectId: string, 
    filePath: string, 
    userData: { user_name: string; session_id: string }
  ) => {
    try {
      setLoading(true);
      setError(null);
      const result = await apiClient.lockFile(projectId, filePath, userData);
      return result;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to lock file';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const unlockFile = useCallback(async (projectId: string, filePath: string) => {
    try {
      setLoading(true);
      setError(null);
      const result = await apiClient.unlockFile(projectId, filePath);
      return result;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to unlock file';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { lockFile, unlockFile, loading, error };
}

export function useFileUpload() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [progress, setProgress] = useState(0);

  const uploadFile = useCallback(async (
    projectId: string, 
    file: File, 
    filePath?: string
  ) => {
    try {
      setLoading(true);
      setError(null);
      setProgress(0);

      // For now, we don't have upload progress from the API
      // You can implement this with XMLHttpRequest if needed
      const result = await apiClient.uploadFile(projectId, file, filePath);
      setProgress(100);
      return result;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to upload file';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { uploadFile, loading, error, progress };
}

export function usePresenceUpdate() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updatePresence = useCallback(async (
    projectId: string,
    data: { user_name: string; status: string; current_file: string }
  ) => {
    try {
      setLoading(true);
      setError(null);
      const result = await apiClient.updatePresence(projectId, data);
      return result;
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update presence';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { updatePresence, loading, error };
}

// Utility hook for health checks
export function useHealthCheck() {
  const [isHealthy, setIsHealthy] = useState<boolean | null>(null);
  const [lastCheck, setLastCheck] = useState<Date | null>(null);

  const checkHealth = useCallback(async () => {
    try {
      await apiClient.healthCheck();
      setIsHealthy(true);
      setLastCheck(new Date());
    } catch (err) {
      setIsHealthy(false);
      setLastCheck(new Date());
      console.error('Health check failed:', err);
    }
  }, []);

  useEffect(() => {
    checkHealth();
    
    // Check health every 30 seconds
    const interval = setInterval(checkHealth, 30000);
    return () => clearInterval(interval);
  }, [checkHealth]);

  return { isHealthy, lastCheck, checkHealth };
}

// Hook for managing multiple projects
export function useDashboardData() {
  const { data: projectsData, loading: projectsLoading, error: projectsError } = useProjects();
  const { data: storageData, loading: storageLoading, error: storageError } = useStorageStats();
  const { isHealthy } = useHealthCheck();

  // Combine all project activity feeds
  const [allActivity, setAllActivity] = useState<ActivityEvent[]>([]);
  const [activityLoading, setActivityLoading] = useState(false);

  useEffect(() => {
    if (!projectsData?.projects?.length) return;

    const fetchAllActivity = async () => {
      setActivityLoading(true);
      try {
        const activityPromises = projectsData.projects.map(project =>
          apiClient.getActivityFeed(project.id, 10)
        );
        
        const results = await Promise.allSettled(activityPromises);
        
        const allEvents = results
          .filter((result): result is PromiseFulfilledResult<{ success: boolean; activities: ActivityEvent[] }> => 
            result.status === 'fulfilled' && result.value.success
          )
          .flatMap(result => result.value.activities)
          .sort((a, b) => new Date(b.event_time).getTime() - new Date(a.event_time).getTime())
          .slice(0, 20); // Keep latest 20 events

        setAllActivity(allEvents);
      } catch (err) {
        console.error('Failed to fetch activity:', err);
      } finally {
        setActivityLoading(false);
      }
    };

    fetchAllActivity();
  }, [projectsData?.projects]);

  return {
    projects: projectsData?.projects || [],
    projectsLoading,
    projectsError,
    storageStats: storageData?.stats as StorageStats | undefined,
    storageLoading,
    storageError,
    recentActivity: allActivity,
    activityLoading,
    isHealthy,
    loading: projectsLoading || storageLoading || activityLoading,
    error: projectsError || storageError
  };
}