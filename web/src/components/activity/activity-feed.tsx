// components/activity/activity-feed.tsx
"use client";

import React, { useState, useEffect } from 'react';
import { 
  Activity, 
  GitCommit, 
  Upload, 
  Lock, 
  Unlock, 
  UserPlus, 
  UserMinus,
  FileText,
  AlertCircle,
  Clock
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { ScrollArea } from '@/components/ui/scroll-area';
import { useMyPresence, useOthers } from '@liveblocks/react/suspense';
import { useVCSClient } from '@/lib/vcs-client';
import { WebSocketMessage } from '@/lib/vcs-client';
import { cn } from '@/lib/utils';

// Type definitions for Liveblocks
interface Presence {
  currentFile?: string;
  editingFile?: string;
}

interface UserInfo {
  name?: string;
  image?: string;
}

interface ActivityFeedProps {
  projectId: string;
  limit?: number;
  showLiveActivity?: boolean;
}

interface ActivityItem {
  id: string;
  type: 'commit' | 'file_upload' | 'file_lock' | 'file_unlock' | 'member_join' | 'member_leave' | 'file_edit';
  user_name: string;
  user_avatar?: string;
  message: string;
  file_path?: string;
  commit_hash?: string;
  timestamp: string;
  metadata?: Record<string, any>;
}

export default function ActivityFeed({ 
  projectId, 
  limit = 50, 
  showLiveActivity = true 
}: ActivityFeedProps) {
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [wsConnected, setWsConnected] = useState(false);
  
  // Liveblocks for real-time collaboration
  const [myPresence, updateMyPresence] = useMyPresence();
  const others = useOthers();
  const vcsClient = useVCSClient();

  // Fetch activity history from backend
  const fetchActivityHistory = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/v1/analytics/activity/${projectId}?limit=${limit}`);
      
      if (response.status === 404) {
        setError('Activity feed API not implemented yet. Backend endpoint needed: GET /api/v1/analytics/activity/:project');
        // Show some mock live activity instead
        setActivities([]);
        return;
      }

      if (!response.ok) throw new Error('Failed to fetch activity');
      
      const data = await response.json();
      setActivities(data.activities || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load activity feed');
    } finally {
      setLoading(false);
    }
  };

  // Add new activity item
  const addActivity = (activity: ActivityItem) => {
    setActivities(prev => {
      const updated = [activity, ...prev.slice(0, limit - 1)];
      return updated;
    });
  };

  // Set up WebSocket for real-time updates
  useEffect(() => {
    fetchActivityHistory();

    if (!showLiveActivity) return;

    const ws = vcsClient.connectWebSocket(projectId, (message: WebSocketMessage) => {
      const activity: ActivityItem = {
        id: `${Date.now()}-${Math.random()}`,
        type: message.type as ActivityItem['type'],
        user_name: message.user_name,
        message: message.message || formatActivityMessage(message),
        file_path: message.file_path,
        timestamp: message.timestamp,
        metadata: message.data
      };
      
      addActivity(activity);
      setWsConnected(true);
    });

    return () => {
      ws.close();
      setWsConnected(false);
    };
  }, [projectId, limit, showLiveActivity]);

  // Format WebSocket messages into readable activity
  const formatActivityMessage = (message: WebSocketMessage): string => {
    switch (message.type) {
      case 'file_locked':
        return `locked ${message.file_path}`;
      case 'file_unlocked':
        return `unlocked ${message.file_path}`;
      case 'file_uploaded':
        return `uploaded ${message.file_path}`;
      case 'member_joined':
        return `joined the project`;
      case 'member_left':
        return `left the project`;
      case 'commit_created':
        return `created commit ${message.data?.hash?.substring(0, 7)}`;
      default:
        return `performed ${message.type}`;
    }
  };

  // Get activity icon
  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'commit': return <GitCommit className="w-4 h-4 text-blue-500" />;
      case 'file_upload': return <Upload className="w-4 h-4 text-green-500" />;
      case 'file_lock': return <Lock className="w-4 h-4 text-orange-500" />;
      case 'file_unlock': return <Unlock className="w-4 h-4 text-gray-500" />;
      case 'member_join': return <UserPlus className="w-4 h-4 text-blue-500" />;
      case 'member_leave': return <UserMinus className="w-4 h-4 text-red-500" />;
      case 'file_edit': return <FileText className="w-4 h-4 text-purple-500" />;
      default: return <Activity className="w-4 h-4 text-gray-500" />;
    }
  };

  // Format timestamp
  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (minutes < 1) return 'just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString();
  };

  return (
    <Card className="h-full flex flex-col">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Activity className="w-5 h-5" />
            Activity Feed
          </CardTitle>
          <div className="flex items-center gap-2">
            {showLiveActivity && (
              <Badge variant={wsConnected ? "success" : "secondary"} className="text-xs">
                <div className={cn(
                  "w-2 h-2 rounded-full mr-1",
                  wsConnected ? "bg-green-500 animate-pulse" : "bg-gray-400"
                )} />
                {wsConnected ? 'Live' : 'Offline'}
              </Badge>
            )}
            <Badge variant="outline" className="text-xs">
              {others.length + 1} online
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="flex-1 overflow-hidden p-0">
        {error && (
          <div className="p-4">
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertDescription className="text-sm">{error}</AlertDescription>
            </Alert>
          </div>
        )}

        {/* Live Presence */}
        {showLiveActivity && others.length > 0 && (
          <div className="px-4 py-2 border-b bg-muted/30">
            <h4 className="text-sm font-medium mb-2">Currently Active</h4>
            <div className="space-y-1">
              {others.map((user) => (
                <div key={user.connectionId} className="flex items-center gap-2 text-sm">
                  <Avatar className="w-6 h-6">
                    <AvatarImage src={(user.info as UserInfo)?.image} />
                    <AvatarFallback className="text-xs">
                      {(user.info as UserInfo)?.name?.charAt(0) || 'U'}
                    </AvatarFallback>
                  </Avatar>
                  <span className="font-medium">{(user.info as UserInfo)?.name || 'Anonymous'}</span>
                  {(user.presence as Presence)?.currentFile && (
                    <span className="text-muted-foreground text-xs">
                      viewing {(user.presence as Presence).currentFile?.split('/').pop()}
                    </span>
                  )}
                  {(user.presence as Presence)?.editingFile && (
                    <span className="text-muted-foreground text-xs">
                      editing {(user.presence as Presence).editingFile?.split('/').pop()}
                    </span>
                  )}
                  <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse ml-auto" />
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Activity List */}
        <ScrollArea className="flex-1">
          <div className="p-4 space-y-3">
            {loading ? (
              <div className="flex items-center justify-center py-8">
                <div className="text-muted-foreground">Loading activity...</div>
              </div>
            ) : activities.length > 0 ? (
              activities.map((activity, index) => (
                <div key={activity.id} className="flex items-start gap-3">
                  <div className="flex flex-col items-center">
                    <Avatar className="w-8 h-8">
                      {activity.user_avatar ? (
                        <AvatarImage src={activity.user_avatar} />
                      ) : null}
                      <AvatarFallback className="text-xs">
                        {activity.user_name.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    {index < activities.length - 1 && (
                      <div className="w-px h-6 bg-border mt-2"></div>
                    )}
                  </div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      {getActivityIcon(activity.type)}
                      <span className="font-medium text-sm">{activity.user_name}</span>
                      <span className="text-muted-foreground text-sm">{activity.message}</span>
                      <span className="text-xs text-muted-foreground ml-auto">
                        {formatTime(activity.timestamp)}
                      </span>
                    </div>

                    {activity.file_path && (
                      <div className="text-xs text-muted-foreground font-mono bg-muted px-2 py-1 rounded mt-1 inline-block">
                        {activity.file_path}
                      </div>
                    )}

                    {activity.commit_hash && (
                      <div className="text-xs text-muted-foreground font-mono bg-muted px-2 py-1 rounded mt-1 inline-block">
                        {activity.commit_hash}
                      </div>
                    )}

                    {activity.metadata?.files_count && (
                      <Badge variant="outline" className="text-xs mt-1">
                        {activity.metadata.files_count} files
                      </Badge>
                    )}
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center text-muted-foreground py-8">
                <Activity className="w-8 h-8 mx-auto mb-2 opacity-50" />
                <p>No recent activity</p>
                <p className="text-xs">Activity will appear here as team members work</p>
              </div>
            )}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}