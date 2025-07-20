class VCSClient {
  private baseUrl: string;
  private token?: string;

  constructor(baseUrl: string = '/api/v1') {
    this.baseUrl = baseUrl;
  }

  setAuthToken(token: string) {
    this.token = token;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.token) {
      (headers as Record<string, string>).Authorization = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error(`API endpoint not implemented: ${endpoint}`);
        }
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      return await response.json();
    } catch (error) {
      console.error(`VCS API Error [${endpoint}]:`, error);
      throw error;
    }
  }

  // Project methods
  async getProject(projectId: string) {
    return this.request(`/projects/${projectId}`);
  }

  async getProjectFiles(projectId: string) {
    return this.request(`/projects/${projectId}/files`);
  }

  async getProjectMembers(projectId: string) {
    return this.request(`/projects/${projectId}/members`);
  }

  // File methods
  async getFileContent(projectId: string, filePath: string) {
    const response = await fetch(`${this.baseUrl}/files/${projectId}/${encodeURIComponent(filePath)}`, {
      headers: this.token ? { Authorization: `Bearer ${this.token}` } : {},
    });

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error('File content API not implemented yet');
      }
      throw new Error(`Failed to fetch file: ${response.statusText}`);
    }

    return await response.text();
  }

  async uploadFiles(projectId: string, files: FileList) {
    const formData = new FormData();
    Array.from(files).forEach(file => {
      formData.append('files', file);
    });
    formData.append('project_id', projectId);

    const response = await fetch(`${this.baseUrl}/files/upload`, {
      method: 'POST',
      headers: this.token ? { Authorization: `Bearer ${this.token}` } : {},
      body: formData,
    });

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error('File upload API not implemented yet');
      }
      throw new Error(`Upload failed: ${response.statusText}`);
    }

    return await response.json();
  }

  // Lock methods
  async getFileLocks(projectId: string) {
    return this.request(`/locks/${projectId}`);
  }

  async lockFile(projectId: string, filePath: string, userData: { user_name: string; session_id: string }) {
    return this.request(`/locks/${projectId}/${encodeURIComponent(filePath)}`, {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }

  async unlockFile(projectId: string, filePath: string) {
    return this.request(`/locks/${projectId}/${encodeURIComponent(filePath)}`, {
      method: 'DELETE',
    });
  }

  // FIXED: Real-time WebSocket connection with correct route
  connectWebSocket(projectId: string, onMessage: (data: any) => void): WebSocket {
    const wsUrl = `ws://localhost:8080/api/v1/collaboration/ws?project_id=${projectId}`;
    const ws = new WebSocket(wsUrl);

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        onMessage(data);
      } catch (error) {
        console.error('WebSocket message parsing error:', error);
      }
    };

    ws.onerror = (error) => {
      console.log('WebSocket connection failed. Real-time updates not available.');
    };

    ws.onclose = () => {
      console.log('WebSocket connection closed.');
    };

    ws.onopen = () => {
      console.log('WebSocket connected to project:', projectId);
    };

    return ws;
  }
}

// Create singleton instance
export const vcsClient = new VCSClient();

// Hook for using VCS client in React components
import { useSession } from 'next-auth/react';
import { useEffect } from 'react';

export function useVCSClient() {
  const { data: session } = useSession();

  useEffect(() => {
    if (session?.accessToken) {
      vcsClient.setAuthToken(session.accessToken as string);
    }
  }, [session]);

  return vcsClient;
}