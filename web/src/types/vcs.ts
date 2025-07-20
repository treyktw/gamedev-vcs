export interface FileTreeItem {
  path: string;
  name: string;
  type: 'file' | 'directory';
  size?: number;
  modified_at?: string;
  content_hash?: string;
  is_locked?: boolean;
  locked_by?: string;
  locked_at?: string;
  children?: FileTreeItem[];
}

export interface FileLock {
  file_path: string;
  user_id: string;
  user_name: string;
  locked_at: string;
  session_id: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  is_private: boolean;
  default_branch: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export interface TeamMember {
  id: string;
  name: string;
  email: string;
  image?: string;
  role: 'owner' | 'admin' | 'write' | 'read';
  status: 'online' | 'away' | 'offline';
  current_file?: string;
  last_seen: string;
}

export interface Commit {
  id: string;
  hash: string;
  message: string;
  author: string;
  author_email: string;
  created_at: string;
  branch: string;
  parent_hashes: string[];
  files_changed: number;
  insertions: number;
  deletions: number;
}

export interface WebSocketMessage {
  type: 'file_locked' | 'file_unlocked' | 'file_uploaded' | 'member_joined' | 'member_left' | 'commit_created';
  project_id: string;
  user_id: string;
  user_name: string;
  file_path?: string;
  message?: string;
  timestamp: string;
  data?: any;
}

// Liveblocks types
export interface LiveblocksPresence {
  cursor: { x: number; y: number } | null;
  currentFile: string | null;
  editingFile: string | null;
  user: {
    id: string;
    name: string;
    avatar?: string;
  };
}

export interface LiveblocksStorage {
  fileLocks: Record<string, {
    userId: string;
    userName: string;
    lockedAt: number;
  }>;
  recentActivity: Array<{
    id: string;
    type: string;
    message: string;
    timestamp: number;
    userId: string;
  }>;
}