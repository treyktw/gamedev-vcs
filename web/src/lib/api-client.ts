// lib/api-client.ts - API client for VCS backend
'use client';

const API_BASE_URL = process.env.NEXT_PUBLIC_VCS_API_URL || 'http://localhost:8080';

interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  created_at: string;
  updated_at: string;
}

export interface StorageStats {
  totalFiles: number;
  totalSize: number;
  projects: number;
}

interface LockInfo {
  file_path: string;
  user_name: string;
  session_id: string;
  locked_at: string;
}

interface PresenceInfo {
  user_name: string;
  session_id: string;
  status: string;
  current_file: string;
  last_seen: string;
}

export interface ActivityEvent {
  id: string;
  event_type: string;
  user_name: string;
  file_path?: string;
  event_time: string;
  details?: Record<string, any>;
}

class ApiClient {
  private baseURL: string;
  private authToken?: string;

  constructor(baseURL: string = API_BASE_URL) {
    this.baseURL = baseURL.replace(/\/$/, '');
  }

  setAuthToken(token: string) {
    this.authToken = token;
  }

  private async makeRequest<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options.headers as Record<string, string>,
    };

    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(`API Error: ${response.status} - ${errorData.error || response.statusText}`);
    }

    return response.json();
  }

  // Health check
  async healthCheck(): Promise<{ success: boolean }> {
    return this.makeRequest('/api/v1/health');
  }

  // Projects
  async getProjects(): Promise<{ success: boolean; projects: Project[] }> {
    return this.makeRequest('/api/v1/projects');
  }

  // Storage stats
  async getStorageStats(): Promise<{ success: boolean; stats: StorageStats }> {
    return this.makeRequest('/api/v1/system/storage/stats');
  }

  // Project locks
  async getProjectLocks(projectId: string): Promise<{ success: boolean; locks: LockInfo[] }> {
    return this.makeRequest(`/api/v1/locks/${projectId}`);
  }

  // Project presence
  async getProjectPresence(projectId: string): Promise<{ success: boolean; presence: PresenceInfo[] }> {
    return this.makeRequest(`/api/v1/collaboration/${projectId}/presence`);
  }

  // Activity feed
  async getActivityFeed(projectId: string, limit: number = 50): Promise<{ success: boolean; activities: ActivityEvent[] }> {
    return this.makeRequest(`/api/v1/analytics/activity/${projectId}?limit=${limit}`);
  }

  // File operations
  async lockFile(
    projectId: string,
    filePath: string,
    userData: { user_name: string; session_id: string }
  ): Promise<{ success: boolean; locked: boolean; lock_info?: LockInfo }> {
    return this.makeRequest(`/api/v1/locks/${projectId}/${encodeURIComponent(filePath)}`, {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }

  async unlockFile(projectId: string, filePath: string): Promise<{ success: boolean }> {
    return this.makeRequest(`/api/v1/locks/${projectId}/${encodeURIComponent(filePath)}`, {
      method: 'DELETE',
    });
  }

  async uploadFile(
    projectId: string,
    file: File,
    filePath?: string
  ): Promise<{ success: boolean; content_hash: string; size: number; file_path: string }> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('project_id', projectId);
    if (filePath) {
      formData.append('file_path', filePath);
    }

    const url = `${this.baseURL}/api/v1/files/upload?project=${projectId}`;
    const headers: Record<string, string> = {};
    
    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: formData,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(`Upload failed: ${response.status} - ${errorData.error || response.statusText}`);
    }

    return response.json();
  }

  // Presence updates
  async updatePresence(
    projectId: string,
    data: { user_name: string; status: string; current_file: string }
  ): Promise<{ success: boolean }> {
    return this.makeRequest(`/api/v1/collaboration/${projectId}/presence`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }
}

// Export singleton instance
export const apiClient = new ApiClient(); 