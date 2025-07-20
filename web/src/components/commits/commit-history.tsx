// components/commits/commit-history.tsx
"use client";

import React, { useState, useEffect } from 'react';
import { GitCommit, Calendar, User, FileText, Plus, Minus, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { CodeComparison } from '@/components/magicui/code-comparison';
import { useVCSClient } from '@/lib/vcs-client';
import { Commit } from '@/types/vcs';

import { cn } from '@/lib/utils';

const getChangeIcon = (changeType: string) => {
  switch (changeType) {
    case 'added': return <Plus className="w-3 h-3 text-green-500" />;
    case 'deleted': return <Minus className="w-3 h-3 text-red-500" />;
    case 'modified': return <FileText className="w-3 h-3 text-blue-500" />;
    default: return <FileText className="w-3 h-3 text-gray-500" />;
  }
};

interface CommitHistoryProps {
  projectId: string;
  branch?: string;
}

interface CommitWithFiles extends Commit {
  files: Array<{
    path: string;
    change_type: 'added' | 'modified' | 'deleted' | 'moved';
    insertions: number;
    deletions: number;
    before_content?: string;
    after_content?: string;
  }>;
}

export default function CommitHistory({ projectId, branch = 'main' }: CommitHistoryProps) {
  const [commits, setCommits] = useState<Commit[]>([]);
  const [selectedCommit, setSelectedCommit] = useState<CommitWithFiles | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const vcsClient = useVCSClient();

  // Fetch commits
  const fetchCommits = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/v1/commits/${projectId}?branch=${branch}`);
      
      if (response.status === 404) {
        setError('Commit history API not implemented yet. Backend endpoint needed: GET /api/v1/commits/:project');
        return;
      }

      if (!response.ok) throw new Error('Failed to fetch commits');
      
      const data = await response.json();
      setCommits(data.commits || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load commit history');
    } finally {
      setLoading(false);
    }
  };

  // Fetch commit details with file changes
  const fetchCommitDetails = async (commitHash: string) => {
    try {
      const response = await fetch(`/api/v1/commits/${projectId}/${commitHash}`);
      
      if (response.status === 404) {
        setError('Commit details API not implemented yet. Backend endpoint needed: GET /api/v1/commits/:project/:hash');
        return;
      }

      if (!response.ok) throw new Error('Failed to fetch commit details');
      
      const data = await response.json();
      setSelectedCommit(data.commit);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load commit details');
    }
  };

  useEffect(() => {
    fetchCommits();
  }, [projectId, branch]);

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-muted-foreground">Loading commit history...</div>
      </div>
    );
  }

  return (
    <div className="flex h-full">
      {/* Commit List */}
      <div className="w-1/2 border-r overflow-auto">
        <div className="p-4 border-b">
          <h2 className="font-semibold text-lg">Commit History</h2>
          <p className="text-sm text-muted-foreground">Branch: {branch}</p>
        </div>

        {error && (
          <div className="p-4">
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          </div>
        )}

        <div className="p-4 space-y-4">
          {commits.length > 0 ? (
            commits.map((commit, index) => (
              <Card 
                key={commit.hash}
                className={cn(
                  "cursor-pointer transition-colors hover:bg-muted/50",
                  selectedCommit?.hash === commit.hash && "border-primary bg-primary/5"
                )}
                onClick={() => fetchCommitDetails(commit.hash)}
              >
                <CardContent className="p-4">
                  <div className="flex items-start gap-3">
                    <div className="flex flex-col items-center">
                      <Avatar className="w-8 h-8">
                        <AvatarFallback className="text-xs">
                          {commit.author.charAt(0).toUpperCase()}
                        </AvatarFallback>
                      </Avatar>
                      {index < commits.length - 1 && (
                        <div className="w-px h-8 bg-border mt-2"></div>
                      )}
                    </div>
                    
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <GitCommit className="w-4 h-4 text-muted-foreground" />
                        <code className="text-xs bg-muted px-2 py-1 rounded">
                          {commit.hash.substring(0, 7)}
                        </code>
                      </div>
                      
                      <h3 className="font-medium text-sm mb-2 line-clamp-2">
                        {commit.message}
                      </h3>
                      
                      <div className="flex items-center gap-4 text-xs text-muted-foreground">
                        <div className="flex items-center gap-1">
                          <User className="w-3 h-3" />
                          {commit.author}
                        </div>
                        <div className="flex items-center gap-1">
                          <Calendar className="w-3 h-3" />
                          {formatDate(commit.created_at)}
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-2 mt-2">
                        <Badge variant="outline" className="text-xs">
                          {commit.files_changed} files
                        </Badge>
                        {commit.insertions > 0 && (
                          <Badge variant="outline" className="text-green-600 text-xs">
                            +{commit.insertions}
                          </Badge>
                        )}
                        {commit.deletions > 0 && (
                          <Badge variant="outline" className="text-red-600 text-xs">
                            -{commit.deletions}
                          </Badge>
                        )}
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))
          ) : (
            <div className="text-center text-muted-foreground py-8">
              No commits found in this branch
            </div>
          )}
        </div>
      </div>

      {/* Commit Details */}
      <div className="w-1/2 overflow-auto">
        {selectedCommit ? (
          <CommitDetails commit={selectedCommit} />
        ) : (
          <div className="flex items-center justify-center h-64">
            <div className="text-center text-muted-foreground">
              <GitCommit className="w-12 h-12 mx-auto mb-4 opacity-50" />
              <p>Select a commit to view details</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// Commit Details Component
function CommitDetails({ commit }: { commit: CommitWithFiles }) {
  const [selectedFile, setSelectedFile] = useState<string | null>(null);

  const selectedFileData = selectedFile 
    ? commit.files.find(f => f.path === selectedFile)
    : null;

  return (
    <div className="h-full flex flex-col">
      {/* Commit Header */}
      <div className="p-4 border-b">
        <div className="flex items-center gap-2 mb-2">
          <GitCommit className="w-5 h-5" />
          <h3 className="font-semibold">{commit.message}</h3>
        </div>
        
        <div className="flex items-center gap-4 text-sm text-muted-foreground mb-3">
          <div className="flex items-center gap-2">
            <Avatar className="w-6 h-6">
              <AvatarFallback className="text-xs">
                {commit.author.charAt(0).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <span>{commit.author}</span>
          </div>
          <span>{new Date(commit.created_at).toLocaleString()}</span>
        </div>

        <div className="flex items-center gap-2">
          <code className="text-xs bg-muted px-2 py-1 rounded">
            {commit.hash}
          </code>
          <Badge variant="outline">{commit.branch}</Badge>
        </div>
      </div>

      {/* Files Changed */}
      <div className="flex-1 overflow-hidden flex flex-col">
        <div className="p-4 border-b">
          <h4 className="font-medium mb-2">Files Changed ({commit.files?.length || 0})</h4>
          <div className="space-y-1 max-h-32 overflow-auto">
            {commit.files?.map((file) => (
              <div
                key={file.path}
                className={cn(
                  "flex items-center gap-2 p-2 rounded cursor-pointer hover:bg-muted/50",
                  selectedFile === file.path && "bg-primary/10 border border-primary/20"
                )}
                onClick={() => setSelectedFile(file.path)}
              >
                {getChangeIcon(file.change_type)}
                <span className="flex-1 text-sm font-mono">{file.path}</span>
                <div className="flex items-center gap-1 text-xs">
                  {file.insertions > 0 && (
                    <span className="text-green-600">+{file.insertions}</span>
                  )}
                  {file.deletions > 0 && (
                    <span className="text-red-600">-{file.deletions}</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* File Diff */}
        <div className="flex-1 overflow-auto">
          {selectedFileData ? (
            <div className="p-4">
              {selectedFileData.before_content && selectedFileData.after_content ? (
                <CodeComparison
                  beforeCode={selectedFileData.before_content}
                  afterCode={selectedFileData.after_content}
                  language="typescript"
                  filename={selectedFileData.path.split('/').pop() || 'file'}
                  lightTheme="github-light"
                  darkTheme="github-dark"
                />
              ) : (
                <div className="text-center text-muted-foreground py-8">
                  <FileText className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p>File diff not available</p>
                  <p className="text-xs">
                    {selectedFileData.change_type === 'added' && 'New file added'}
                    {selectedFileData.change_type === 'deleted' && 'File was deleted'}
                    {selectedFileData.change_type === 'moved' && 'File was moved'}
                  </p>
                </div>
              )}
            </div>
          ) : (
            <div className="flex items-center justify-center h-64">
              <div className="text-center text-muted-foreground">
                <FileText className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p>Select a file to view changes</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}