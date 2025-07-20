// components/team/team-management.tsx
"use client";

import React, { useState, useEffect } from 'react';
import { 
  Users, 
  UserPlus, 
  Settings, 
  Mail, 
  Crown, 
  Shield, 
  Edit, 
  Trash2,
  Circle,
  AlertCircle,
  Clock
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useOthers, useMyPresence } from '@liveblocks/react/suspense';
import { useVCSClient } from '@/lib/vcs-client';
import { TeamMember } from '@/types/vcs';

// Type definitions for Liveblocks
interface Presence {
  currentFile?: string;
  editingFile?: string;
}

interface UserInfo {
  id?: string;
  name?: string;
  avatar?: string;
}

interface TeamManagementProps {
  projectId: string;
}

interface InviteData {
  email: string;
  role: 'read' | 'write' | 'admin';
  message?: string;
}

export default function TeamManagement({ projectId }: TeamManagementProps) {
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false);
  const [inviteData, setInviteData] = useState<InviteData>({
    email: '',
    role: 'read'
  });
  const [inviting, setInviting] = useState(false);

  // Liveblocks for real-time presence
  const others = useOthers();

  // Fetch team members
  const fetchMembers = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/v1/projects/${projectId}/members`);
      
      if (response.status === 404) {
        setError('Team management API not implemented yet. Backend endpoint needed: GET /api/v1/projects/:id/members');
        // Show mock data for demonstration
        setMembers([
          {
            id: '1',
            name: 'Current User',
            email: 'user@example.com',
            role: 'owner',
            status: 'online',
            last_seen: new Date().toISOString()
          }
        ]);
        return;
      }

      if (!response.ok) throw new Error('Failed to fetch team members');
      
      const data = await response.json();
      setMembers(data.members || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load team members');
    } finally {
      setLoading(false);
    }
  };

  // Invite new member
  const inviteMember = async () => {
    try {
      setInviting(true);
      const response = await fetch(`/api/v1/projects/${projectId}/members/invite`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(inviteData)
      });

      if (response.status === 404) {
        setError('Member invitation API not implemented yet. Backend endpoint needed: POST /api/v1/projects/:id/members/invite');
        return;
      }

      if (!response.ok) throw new Error('Failed to send invitation');

      await fetchMembers();
      setInviteDialogOpen(false);
      setInviteData({ email: '', role: 'read' });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to invite member');
    } finally {
      setInviting(false);
    }
  };

  // Update member role
  const updateMemberRole = async (memberId: string, newRole: string) => {
    try {
      const response = await fetch(`/api/v1/projects/${projectId}/members/${memberId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role: newRole })
      });

      if (response.status === 404) {
        setError('Member role update API not implemented yet. Backend endpoint needed: PATCH /api/v1/projects/:id/members/:memberId');
        return;
      }

      if (!response.ok) throw new Error('Failed to update member role');

      await fetchMembers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update member role');
    }
  };

  // Remove member
  const removeMember = async (memberId: string) => {
    try {
      const response = await fetch(`/api/v1/projects/${projectId}/members/${memberId}`, {
        method: 'DELETE'
      });

      if (response.status === 404) {
        setError('Member removal API not implemented yet. Backend endpoint needed: DELETE /api/v1/projects/:id/members/:memberId');
        return;
      }

      if (!response.ok) throw new Error('Failed to remove member');

      await fetchMembers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove member');
    }
  };

  useEffect(() => {
    fetchMembers();
  }, [projectId]);

  // Get role icon
  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'owner': return <Crown className="w-4 h-4 text-yellow-500" />;
      case 'admin': return <Shield className="w-4 h-4 text-blue-500" />;
      case 'write': return <Edit className="w-4 h-4 text-green-500" />;
      case 'read': return <Users className="w-4 h-4 text-gray-500" />;
      default: return <Users className="w-4 h-4 text-gray-500" />;
    }
  };

  // Get status indicator
  const getStatusIndicator = (status: string, lastSeen: string) => {
    const lastSeenDate = new Date(lastSeen);
    const now = new Date();
    const diffMinutes = Math.floor((now.getTime() - lastSeenDate.getTime()) / 60000);

    if (status === 'online' || diffMinutes < 5) {
      return <Circle className="w-3 h-3 text-green-500 fill-current" />;
    } else if (diffMinutes < 30) {
      return <Circle className="w-3 h-3 text-yellow-500 fill-current" />;
    } else {
      return <Circle className="w-3 h-3 text-gray-400 fill-current" />;
    }
  };

  // Format last seen
  const formatLastSeen = (lastSeen: string) => {
    const date = new Date(lastSeen);
    const now = new Date();
    const diffMinutes = Math.floor((now.getTime() - date.getTime()) / 60000);

    if (diffMinutes < 1) return 'Active now';
    if (diffMinutes < 60) return `${diffMinutes}m ago`;
    if (diffMinutes < 1440) return `${Math.floor(diffMinutes / 60)}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Team Members</h2>
          <p className="text-muted-foreground">
            Manage team access and permissions
          </p>
        </div>
        
        <Dialog open={inviteDialogOpen} onOpenChange={setInviteDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <UserPlus className="w-4 h-4 mr-2" />
              Invite Member
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Invite Team Member</DialogTitle>
              <DialogDescription>
                Send an invitation to join this project
              </DialogDescription>
            </DialogHeader>
            
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">Email Address</label>
                <Input
                  type="email"
                  placeholder="colleague@example.com"
                  value={inviteData.email}
                  onChange={(e) => setInviteData(prev => ({ ...prev, email: e.target.value }))}
                />
              </div>
              
              <div>
                <label className="text-sm font-medium">Role</label>
                <Select
                  value={inviteData.role}
                  onValueChange={(value) => setInviteData(prev => ({ ...prev, role: value as any }))}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="read">Read - Can view files</SelectItem>
                    <SelectItem value="write">Write - Can edit files</SelectItem>
                    <SelectItem value="admin">Admin - Can manage team</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              
              <div>
                <label className="text-sm font-medium">Message (Optional)</label>
                <Input
                  placeholder="Welcome to the team!"
                  value={inviteData.message || ''}
                  onChange={(e) => setInviteData(prev => ({ ...prev, message: e.target.value }))}
                />
              </div>
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setInviteDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button
                onClick={inviteMember}
                disabled={!inviteData.email || inviting}
              >
                {inviting ? 'Sending...' : 'Send Invitation'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Error Display */}
      {error && (
        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Team Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Members</p>
                <p className="text-2xl font-bold">{members.length}</p>
              </div>
              <Users className="w-8 h-8 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Online Now</p>
                <p className="text-2xl font-bold">{others.length + 1}</p>
              </div>
              <Circle className="w-8 h-8 text-green-500 fill-current" />
            </div>
          </CardContent>
        </Card>
        
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Active Today</p>
                <p className="text-2xl font-bold">
                  {members.filter(m => {
                    const diff = Date.now() - new Date(m.last_seen).getTime();
                    return diff < 24 * 60 * 60 * 1000; // 24 hours
                  }).length}
                </p>
              </div>
              <Clock className="w-8 h-8 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Members List */}
      <Card>
        <CardHeader>
          <CardTitle>Project Members</CardTitle>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="text-muted-foreground">Loading team members...</div>
            </div>
          ) : (
            <div className="space-y-4">
              {members.map((member) => {
                const isOnlineLive = others.some(other => (other.info as UserInfo)?.id === member.id);
                const actualStatus = isOnlineLive ? 'online' : member.status;
                
                return (
                  <div key={member.id} className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="flex items-center gap-4">
                      <div className="relative">
                        <Avatar className="w-12 h-12">
                          <AvatarImage src={member.image} />
                          <AvatarFallback>{member.name.charAt(0).toUpperCase()}</AvatarFallback>
                        </Avatar>
                        <div className="absolute -bottom-1 -right-1">
                          {getStatusIndicator(actualStatus, member.last_seen)}
                        </div>
                      </div>
                      
                      <div>
                        <div className="flex items-center gap-2">
                          <h3 className="font-medium">{member.name}</h3>
                          {getRoleIcon(member.role)}
                          <Badge variant={member.role === 'owner' ? 'default' : 'secondary'}>
                            {member.role}
                          </Badge>
                        </div>
                        <p className="text-sm text-muted-foreground">{member.email}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatLastSeen(member.last_seen)}
                        </p>
                      </div>
                    </div>

                    {member.role !== 'owner' && (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="sm">
                            <Settings className="w-4 h-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem onClick={() => updateMemberRole(member.id, 'read')}>
                            Change to Read
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => updateMemberRole(member.id, 'write')}>
                            Change to Write
                          </DropdownMenuItem>
                          <DropdownMenuItem onClick={() => updateMemberRole(member.id, 'admin')}>
                            Change to Admin
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem 
                            className="text-red-600"
                            onClick={() => removeMember(member.id)}
                          >
                            <Trash2 className="w-4 h-4 mr-2" />
                            Remove Member
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Live Collaboration Status */}
      {others.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Circle className="w-4 h-4 text-green-500 fill-current animate-pulse" />
              Currently Active
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {others.map((user) => (
                <div key={user.connectionId} className="flex items-center gap-3 p-3 bg-muted/30 rounded-lg">
                  <Avatar className="w-8 h-8">
                    <AvatarImage src={(user.info as UserInfo)?.avatar} />
                    <AvatarFallback>
                      {(user.info as UserInfo)?.name?.charAt(0) || 'U'}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex-1">
                    <p className="font-medium text-sm">{(user.info as UserInfo)?.name || 'Anonymous'}</p>
                    {(user.presence as Presence)?.currentFile ? (
                      <p className="text-xs text-muted-foreground">
                        📁 {(user.presence as Presence).currentFile?.split('/').pop()}
                      </p>
                    ) : (
                      <p className="text-xs text-muted-foreground">
                        🌐 Browsing project
                      </p>
                    )}
                  </div>
                  <Badge variant="outline" className="text-xs bg-green-50">
                    Live
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}