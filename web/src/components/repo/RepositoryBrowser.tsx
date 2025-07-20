'use client';

import React, { useState, useEffect } from 'react';
import { FileIcon, GitBranch, Upload, Search, Lock, Unlock, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Alert, AlertDescription } from '@/components/ui/alert';

import { useOthers, useMyPresence } from '@liveblocks/react/suspense';
import { File, Folder, Tree, TreeViewElement } from '../magicui/file-tree';
import { useSession } from 'next-auth/react';

interface FileTreeItem {
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

interface FileLock {
  file_path: string;
  user_id: string;
  user_name: string;
  locked_at: string;
  session_id: string;
}

interface TeamMember {
  id: string;
  name: string;
  email: string;
  image?: string;
  status: 'online' | 'away' | 'offline';
  current_file?: string;
  last_seen: string;
}

interface RepositoryBrowserProps {
  projectId: string;
}

// Convert backend file tree to TreeViewElement format
function convertToTreeViewElement(item: FileTreeItem): TreeViewElement {
  return {
    id: item.path,
    name: item.name,
    isSelectable: true,
    children: item.children?.map(convertToTreeViewElement)
  };
}

export default function RepositoryBrowser({ projectId }: RepositoryBrowserProps) {
  const [fileTree, setFileTree] = useState<FileTreeItem[]>([]);
  const [fileLocks, setFileLocks] = useState<Record<string, FileLock>>({});
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  
  // Liveblocks presence
  const [myPresence, updateMyPresence] = useMyPresence();
  const others = useOthers();

  const { data: session } = useSession();

  // Fetch file tree from backend
  const fetchFileTree = async () => {
    try {
      const response = await fetch(`/api/v1/projects/${projectId}/files`);
      if (response.status === 404) {
        setError('File tree API not implemented yet. Backend endpoint needed: GET /api/v1/projects/:id/files');
        return;
      }
      if (!response.ok) throw new Error('Failed to fetch files');
      
      const data = await response.json();
      setFileTree(data.files || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load file tree');
    }
  };

  // Fetch file locks
  const fetchFileLocks = async () => {
    try {
      const response = await fetch(`/api/v1/locks/${projectId}`);
      if (response.status === 404) {
        console.log('File locks API not implemented yet. Backend endpoint needed: GET /api/v1/locks/:project');
        return;
      }
      if (!response.ok) throw new Error('Failed to fetch locks');
      
      const data = await response.json();
      const lockMap: Record<string, FileLock> = {};
      data.locks?.forEach((lock: FileLock) => {
        lockMap[lock.file_path] = lock;
      });
      setFileLocks(lockMap);
    } catch (err) {
      console.error('Failed to fetch file locks:', err);
    }
  };

  // Fetch team members
  const fetchTeamMembers = async () => {
    try {
      const response = await fetch(`/api/v1/projects/${projectId}/members`);
      if (response.status === 404) {
        console.log('Team members API not implemented yet. Backend endpoint needed: GET /api/v1/projects/:id/members');
        return;
      }
      if (!response.ok) throw new Error('Failed to fetch team members');
      
      const data = await response.json();
      setTeamMembers(data.members || []);
    } catch (err) {
      console.error('Failed to fetch team members:', err);
    }
  };

  // Lock/unlock file
  const toggleFileLock = async (filePath: string) => {
    try {
      const isLocked = fileLocks[filePath];
      const method = isLocked ? 'DELETE' : 'POST';
      const response = await fetch(`/api/v1/locks/${projectId}/${encodeURIComponent(filePath)}`, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: method === 'POST' ? JSON.stringify({
          user_name: 'Current User', // Replace with actual user
          session_id: 'session-id' // Replace with actual session
        }) : undefined
      });

      if (response.status === 404) {
        setError(`File locking API not implemented yet. Backend endpoint needed: ${method} /api/v1/locks/:project/:file`);
        return;
      }

      if (!response.ok) throw new Error(`Failed to ${isLocked ? 'unlock' : 'lock'} file`);
      
      // Refresh locks
      await fetchFileLocks();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle file lock');
    }
  };

  // Update presence when file is selected
  const handleFileSelect = (filePath: string) => {
    setSelectedFile(filePath);
    updateMyPresence({ currentFile: filePath });
  };

  // Upload files
  const handleFileUpload = async (files: FileList) => {
    try {
      const formData = new FormData();
      Array.from(files).forEach(file => {
        formData.append('files', file);
      });
      formData.append('project_id', projectId);

      const response = await fetch('/api/v1/files/upload', {
        method: 'POST',
        body: formData
      });

      if (response.status === 404) {
        setError('File upload API not implemented yet. Backend endpoint needed: POST /api/v1/files/upload');
        return;
      }

      if (!response.ok) throw new Error('Upload failed');

      // Refresh file tree
      await fetchFileTree();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    }
  };

  // Initialize data
  useEffect(() => {
    const loadData = async () => {
      setLoading(true);
      await Promise.all([
        fetchFileTree(),
        fetchFileLocks(), 
        fetchTeamMembers()
      ]);
      setLoading(false);
    };
    
    loadData();

    // Set up WebSocket for real-time updates
    const ws = new WebSocket(`ws://localhost:8080/api/v1/collaboration/ws?project_id=${projectId}&token=${session?.accessToken || ''}`);
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === 'file_locked' || data.type === 'file_unlocked') {
        fetchFileLocks();
      } else if (data.type === 'file_uploaded') {
        fetchFileTree();
      }
    };
    ws.onerror = () => {
      console.log('WebSocket not available. Backend endpoint needed: WS /api/v1/collaboration/ws');
    };

    return () => ws.close();
  }, [projectId]);

  // Convert file tree for TreeView component
  const treeViewElements = fileTree.map(convertToTreeViewElement);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-muted-foreground">Loading repository...</div>
      </div>
    );
  }

  return (
    <div className="flex h-screen">
      {/* Sidebar - File Tree */}
      <div className="w-80 border-r bg-muted/30 flex flex-col">
        {/* Header */}
        <div className="p-4 border-b">
          <div className="flex items-center justify-between mb-4">
            <h2 className="font-semibold">Repository Files</h2>
            <div className="flex items-center gap-2">
              <Button size="sm" variant="outline">
                <GitBranch className="w-4 h-4 mr-2" />
                main
              </Button>
            </div>
          </div>

          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4" />
            <Input
              placeholder="Search files..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>

          {/* Upload */}
          <div className="mt-3">
            <input
              type="file"
              multiple
              id="file-upload"
              className="hidden"
              onChange={(e) => e.target.files && handleFileUpload(e.target.files)}
            />
            <Button 
              size="sm" 
              variant="outline" 
              className="w-full"
              onClick={() => document.getElementById('file-upload')?.click()}
            >
              <Upload className="w-4 h-4 mr-2" />
              Upload Files
            </Button>
          </div>
        </div>

        {/* Error Display */}
        {error && (
          <div className="p-4">
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription className="text-sm">
                {error}
              </AlertDescription>
            </Alert>
          </div>
        )}

        {/* File Tree */}
        <div className="flex-1 overflow-hidden">
          {treeViewElements.length > 0 ? (
            <Tree
              className="p-2"
              initialSelectedId={selectedFile || undefined}
              elements={treeViewElements}
              indicator={true}
            >
              {treeViewElements.map((element) => (
                <TreeNode
                  key={element.id}
                  element={element}
                  fileLocks={fileLocks}
                  onFileSelect={handleFileSelect}
                  onToggleLock={toggleFileLock}
                />
              ))}
            </Tree>
          ) : (
            <div className="p-4 text-center text-muted-foreground">
              No files found. Upload some files to get started.
            </div>
          )}
        </div>

        {/* Team Presence */}
        <div className="p-4 border-t">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium">Team Online</span>
            <Badge variant="secondary">{others.length + 1}</Badge>
          </div>
          <div className="flex -space-x-2">
            {/* Current user */}
            <Avatar className="w-8 h-8 border-2 border-background">
              <AvatarFallback>You</AvatarFallback>
            </Avatar>
            {/* Other users */}
            {others.map((user) => (
              <Avatar key={user.connectionId} className="w-8 h-8 border-2 border-background">
                <AvatarImage src={(user.info as { image?: string })?.image} />
                <AvatarFallback>{(user.info as { name?: string })?.name?.charAt(0)}</AvatarFallback>
              </Avatar>
            ))}
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {selectedFile ? (
          <FileViewer 
            projectId={projectId}
            filePath={selectedFile}
            isLocked={!!fileLocks[selectedFile]}
            lockedBy={fileLocks[selectedFile]?.user_name}
            onToggleLock={() => toggleFileLock(selectedFile)}
          />
        ) : (
          <div className="flex-1 flex items-center justify-center text-muted-foreground">
            Select a file to view its contents
          </div>
        )}
      </div>
    </div>
  );
}

// Tree Node Component
function TreeNode({ 
  element, 
  fileLocks, 
  onFileSelect, 
  onToggleLock 
}: {
  element: TreeViewElement;
  fileLocks: Record<string, FileLock>;
  onFileSelect: (path: string) => void;
  onToggleLock: (path: string) => void;
}) {
  const isFile = !element.children;
  const isLocked = fileLocks[element.id];

  if (isFile) {
    return (
      <File
        value={element.id}
        className="group"
        onClick={() => onFileSelect(element.id)}
      >
        <span className="flex-1">{element.name}</span>
        {isLocked && (
          <div className="flex items-center gap-1">
            <Lock className="w-3 h-3 text-orange-500" />
            <span className="text-xs text-muted-foreground">{isLocked.user_name}</span>
          </div>
        )}
        <Button
          size="sm"
          variant="ghost"
          className="opacity-0 group-hover:opacity-100 h-6 w-6 p-0"
          onClick={(e) => {
            e.stopPropagation();
            onToggleLock(element.id);
          }}
        >
          {isLocked ? <Unlock className="w-3 h-3" /> : <Lock className="w-3 h-3" />}
        </Button>
      </File>
    );
  }

  return (
    <Folder element={element.name} value={element.id}>
      {element.children?.map((child) => (
        <TreeNode
          key={child.id}
          element={child}
          fileLocks={fileLocks}
          onFileSelect={onFileSelect}
          onToggleLock={onToggleLock}
        />
      ))}
    </Folder>
  );
}

// File Viewer Component  
function FileViewer({
  projectId,
  filePath,
  isLocked,
  lockedBy,
  onToggleLock
}: {
  projectId: string;
  filePath: string;
  isLocked: boolean;
  lockedBy?: string;
  onToggleLock: () => void;
}) {
  const [content, setContent] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchFileContent = async () => {
      try {
        const response = await fetch(`/api/v1/files/${projectId}/${encodeURIComponent(filePath)}`);
        if (response.status === 404) {
          setError('File content API not implemented yet. Backend endpoint needed: GET /api/v1/files/:project/:file');
          return;
        }
        if (!response.ok) throw new Error('Failed to fetch file content');
        
        const text = await response.text();
        setContent(text);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load file');
      } finally {
        setLoading(false);
      }
    };

    fetchFileContent();
  }, [projectId, filePath]);

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-muted-foreground">Loading file...</div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col">
      {/* File Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-2">
          <FileIcon className="w-4 h-4" />
          <span className="font-medium">{filePath.split('/').pop()}</span>
          {isLocked && (
            <Badge variant="outline" className="text-orange-600">
              <Lock className="w-3 h-3 mr-1" />
              Locked by {lockedBy}
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant={isLocked ? "outline" : "default"}
            onClick={onToggleLock}
          >
            {isLocked ? (
              <>
                <Unlock className="w-4 h-4 mr-2" />
                Unlock
              </>
            ) : (
              <>
                <Lock className="w-4 h-4 mr-2" />
                Lock & Edit
              </>
            )}
          </Button>
        </div>
      </div>

      {/* File Content */}
      <div className="flex-1 overflow-auto">
        {error ? (
          <div className="p-4">
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          </div>
        ) : (
          <pre className="p-4 text-sm font-mono whitespace-pre-wrap">
            {content}
          </pre>
        )}
      </div>
    </div>
  );
}