'use client';

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Separator } from "@/components/ui/separator";
import { Progress } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { 
  GitBranch, 
  Folder, 
  Users, 
  Activity, 
  Clock, 
  Lock, 
  Unlock,
  Plus,
  TrendingUp,
  FileText,
  Download,
  Upload,
  AlertCircle,
  ExternalLink,
  FolderOpen
} from "lucide-react";
import Link from "next/link";
import { trpc } from "@/lib/trpc/client";
import { useSession } from "next-auth/react";

function DashboardStats({ projects, storageStats, loading }: {
  projects: any[],
  storageStats: any,
  loading: boolean
}) {
  if (loading) {
    return (
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i} className="border-2">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-4 w-4" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-16 mb-2" />
              <Skeleton className="h-3 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  const totalProjects = projects.length;
  const storageUsed = storageStats ? Math.round(storageStats.totalSize / (1024 * 1024)) : 0; // MB
  const storagePercentage = Math.min((storageUsed / (5 * 1024)) * 100, 100); // 5GB limit
  const activeProjects = projects.filter(p => p.stats?.lastActivity && 
    new Date(p.stats.lastActivity).getTime() > Date.now() - (7 * 24 * 60 * 60 * 1000) // Last 7 days
  ).length;

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card className="border-2">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Total Projects</CardTitle>
          <Folder className="w-4 h-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{totalProjects}</div>
          <p className="text-xs text-muted-foreground">
            {activeProjects} active this week
          </p>
        </CardContent>
      </Card>

      <Card className="border-2">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Storage Used</CardTitle>
          <TrendingUp className="w-4 h-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{storageUsed} MB</div>
          <Progress value={storagePercentage} className="mt-2" />
          <p className="text-xs text-muted-foreground mt-1">
            {Math.round(storagePercentage)}% of 5GB used
          </p>
        </CardContent>
      </Card>

      <Card className="border-2">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Team Members</CardTitle>
          <Users className="w-4 h-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {projects.reduce((acc, p) => acc + (p.stats?.contributors || 1), 0)}
          </div>
          <p className="text-xs text-muted-foreground">
            Across all projects
          </p>
        </CardContent>
      </Card>

      <Card className="border-2">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium">Files Managed</CardTitle>
          <FileText className="w-4 h-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">
            {projects.reduce((acc, p) => acc + (p.stats?.totalFiles || 0), 0)}
          </div>
          <p className="text-xs text-muted-foreground">
            Total files tracked
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

function RecentActivityCard({ projectId, limit = 4 }: { projectId?: string, limit?: number }) {
  const { data: activities, isLoading } = trpc.analytics.activity.useQuery(
    { projectId: projectId || 'all', limit },
    { enabled: !!projectId }
  );

  const getActivityIcon = (eventType: string) => {
    switch (eventType) {
      case 'file_locked': return <Lock className="w-4 h-4 text-red-500" />;
      case 'file_unlocked': return <Unlock className="w-4 h-4 text-green-500" />;
      case 'commit_created': return <GitBranch className="w-4 h-4 text-blue-500" />;
      case 'file_modified': return <Upload className="w-4 h-4 text-purple-500" />;
      case 'project_created': return <Plus className="w-4 h-4 text-green-500" />;
      case 'file_uploaded': return <Upload className="w-4 h-4 text-blue-500" />;
      default: return <Activity className="w-4 h-4 text-muted-foreground" />;
    }
  };

  const getActivityDescription = (activity: any) => {
    switch (activity.event_type) {
      case 'file_locked':
        return `locked ${activity.file_path}`;
      case 'file_unlocked':
        return `unlocked ${activity.file_path}`;
      case 'project_created':
        return `created project`;
      case 'file_uploaded':
        return `uploaded ${activity.file_path}`;
      case 'commit_created':
        return `committed changes`;
      default:
        return activity.event_type.replace('_', ' ');
    }
  };

  if (isLoading) {
    return (
      <Card className="border-2">
        <CardHeader>
          <CardTitle className="flex items-center">
            <Activity className="w-5 h-5 mr-2" />
            Recent Activity
          </CardTitle>
          <CardDescription>Loading team activity...</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center space-x-4 p-3 rounded-lg bg-muted/50">
              <Skeleton className="h-8 w-8 rounded-full" />
              <div className="flex-1">
                <Skeleton className="h-4 w-48 mb-1" />
                <Skeleton className="h-3 w-24" />
              </div>
              <Skeleton className="h-4 w-4" />
            </div>
          ))}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-2">
      <CardHeader>
        <CardTitle className="flex items-center">
          <Activity className="w-5 h-5 mr-2" />
          Recent Activity
        </CardTitle>
        <CardDescription>
          Latest updates from your team
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {activities && activities.length > 0 ? (
          <>
            {activities.slice(0, limit).map((activity, i) => (
              <div key={`${activity.event_id}-${i}`} className="flex items-center space-x-4 p-3 rounded-lg bg-muted/50 hover:bg-muted/70 transition-colors">
                <Avatar className="h-8 w-8">
                  <AvatarFallback>
                    {activity.user_name?.split(' ').map((n: string) => n[0]).join('').toUpperCase() || 'U'}
                  </AvatarFallback>
                </Avatar>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">
                    <span className="font-semibold">{activity.user_name}</span> {getActivityDescription(activity)}
                  </p>
                  <p className="text-xs text-muted-foreground flex items-center">
                    <Clock className="w-3 h-3 mr-1" />
                    {new Date(activity.event_time).toLocaleString()}
                  </p>
                  {activity.project && (
                    <p className="text-xs text-blue-600 font-mono">
                      {activity.project}
                    </p>
                  )}
                </div>
                <div className="flex-shrink-0">
                  {getActivityIcon(activity.event_type)}
                </div>
              </div>
            ))}
            
            <Button variant="outline" className="w-full" asChild>
              <Link href="/dashboard/overview">
                View All Activity
              </Link>
            </Button>
          </>
        ) : (
          <div className="text-center py-8">
            <Activity className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground">No recent activity</p>
            <p className="text-sm text-muted-foreground">Start by uploading files or inviting team members</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function ActiveProjectsCard() {
  const { data: projects, isLoading, error } = trpc.projects.list.useQuery();

  if (isLoading) {
    return (
      <Card className="border-2">
        <CardHeader>
          <CardTitle className="flex items-center">
            <Folder className="w-5 h-5 mr-2" />
            Active Projects
          </CardTitle>
          <CardDescription>Loading projects...</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="p-4 rounded-lg border">
              <div className="flex items-center justify-between mb-2">
                <Skeleton className="h-5 w-32" />
                <Skeleton className="h-5 w-16" />
              </div>
              <Skeleton className="h-4 w-full mb-3" />
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-4">
                  <Skeleton className="h-3 w-8" />
                  <Skeleton className="h-3 w-12" />
                </div>
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card className="border-2">
        <CardHeader>
          <CardTitle className="flex items-center">
            <Folder className="w-5 h-5 mr-2" />
            Active Projects
          </CardTitle>
          <CardDescription>Failed to load projects</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8">
            <AlertCircle className="w-12 h-12 text-destructive mx-auto mb-4" />
            <p className="text-destructive">Unable to load projects</p>
            <p className="text-sm text-muted-foreground">{error.message}</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-2">
      <CardHeader>
        <CardTitle className="flex items-center">
          <Folder className="w-5 h-5 mr-2" />
          Active Projects
        </CardTitle>
        <CardDescription>
          Projects with recent activity
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {projects && projects.length > 0 ? (
          <>
            {projects.slice(0, 3).map((project) => (
              <div key={project.id} className="group p-4 rounded-lg border hover:bg-accent/50 transition-all duration-200 hover:shadow-md">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center space-x-2">
                    <FolderOpen className="w-4 h-4 text-muted-foreground group-hover:text-primary transition-colors" />
                    <Link 
                      href={`/dashboard/${project.id}/files`}
                      className="font-semibold hover:underline text-foreground group-hover:text-primary transition-colors"
                    >
                      {project.name}
                    </Link>
                    <ExternalLink className="w-3 h-3 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity" />
                  </div>
                  <Badge 
                    variant="default"
                    className="text-xs"
                  >
                    active
                  </Badge>
                </div>
                
                <p className="text-sm text-muted-foreground mb-3">
                  {project.description || "No description provided"}
                </p>
                
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <div className="flex items-center space-x-4">
                    <span className="flex items-center">
                      <Users className="w-3 h-3 mr-1" />
                      {project.stats?.contributors || 1}
                    </span>
                    <span className="flex items-center">
                      <FileText className="w-3 h-3 mr-1" />
                      {project.stats?.totalFiles || 0} files
                    </span>
                    {project.stats?.totalSize && (
                      <span>{project.stats.totalSize}</span>
                    )}
                  </div>
                  <span>{project.stats?.lastActivity || 'recently'}</span>
                </div>

                {/* Quick Actions */}
                <div className="flex items-center gap-2 mt-3 opacity-0 group-hover:opacity-100 transition-opacity">
                  <Button size="sm" variant="outline" asChild>
                    <Link href={`/dashboard/${project.id}/files`}>
                      <Folder className="w-3 h-3 mr-1" />
                      Browse Files
                    </Link>
                  </Button>
                  <Button size="sm" variant="outline" asChild>
                    <Link href={`/dashboard/${project.id}/commits`}>
                      <GitBranch className="w-3 h-3 mr-1" />
                      Commits
                    </Link>
                  </Button>
                  <Button size="sm" variant="outline" asChild>
                    <Link href={`/dashboard/${project.id}/team`}>
                      <Users className="w-3 h-3 mr-1" />
                      Team
                    </Link>
                  </Button>
                </div>
              </div>
            ))}
            
            <Button variant="outline" className="w-full" asChild>
              <Link href="/dashboard/repositories">
                View All Repositories
              </Link>
            </Button>
          </>
        ) : (
          <div className="text-center py-8">
            <Folder className="w-12 h-12 text-muted-foreground mx-auto mb-4" />
            <p className="text-muted-foreground mb-2">No projects yet</p>
            <p className="text-sm text-muted-foreground mb-4">Create your first repository to get started</p>
            <Button asChild>
              <Link href="/dashboard/repositories/new">
                <Plus className="w-4 h-4 mr-2" />
                Create Repository
              </Link>
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default function DashboardPage() {
  const { data: session } = useSession();
  const { data: projects, isLoading: projectsLoading, error: projectsError } = trpc.projects.list.useQuery();
  const { data: storageStats, isLoading: storageLoading } = trpc.storage.stats.useQuery();
  const { data: healthCheck } = trpc.health.useQuery();

  // Show error state if tRPC is having issues
  if (projectsError && !projectsLoading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
            <p className="text-muted-foreground">Welcome back to GameDev VCS</p>
          </div>
        </div>

        <Card className="border-2 border-destructive/50">
          <CardContent className="flex items-center space-x-4 p-6">
            <AlertCircle className="w-8 h-8 text-destructive" />
            <div>
              <h3 className="font-semibold text-destructive">Unable to connect to VCS server</h3>
              <p className="text-sm text-muted-foreground">
                Make sure your VCS backend is running and the database is accessible
              </p>
              <p className="text-xs text-muted-foreground mt-1">Error: {projectsError.message}</p>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const isHealthy = healthCheck?.status === 'healthy';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
          <p className="text-muted-foreground">
            Welcome back{session?.user?.name ? `, ${session.user.name}` : ''}! Here's what's happening with your projects.
          </p>
        </div>
        <div className="flex items-center space-x-4">
          {/* Health indicator */}
          <div className="flex items-center space-x-2">
            <div className={`w-2 h-2 rounded-full ${isHealthy ? 'bg-green-500 animate-pulse' : 'bg-red-500'}`} />
            <span className="text-xs text-muted-foreground">
              {isHealthy ? 'Connected' : 'Disconnected'}
            </span>
          </div>
          <Button asChild>
            <Link href="/dashboard/repositories/new">
              <Plus className="w-4 h-4 mr-2" />
              New Repository
            </Link>
          </Button>
        </div>
      </div>

      {/* Stats Grid */}
      <DashboardStats 
        projects={projects || []}
        storageStats={storageStats}
        loading={projectsLoading || storageLoading}
      />

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent Activity */}
        <RecentActivityCard projectId="all" />

        {/* Active Projects */}
        <ActiveProjectsCard />
      </div>

      {/* Quick Actions */}
      <Card className="border-2">
        <CardHeader>
          <CardTitle>Quick Actions</CardTitle>
          <CardDescription>
            Common tasks to get you started
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Button variant="outline" className="h-auto p-4 flex flex-col items-center space-y-2" asChild>
              <Link href="/dashboard/repositories/new">
                <Plus className="w-6 h-6" />
                <span className="text-sm font-medium">New Repository</span>
                <span className="text-xs text-muted-foreground">Start a new project</span>
              </Link>
            </Button>

            <Button variant="outline" className="h-auto p-4 flex flex-col items-center space-y-2">
              <Upload className="w-6 h-6" />
              <span className="text-sm font-medium">Upload Assets</span>
              <span className="text-xs text-muted-foreground">Add files to project</span>
            </Button>

            <Button variant="outline" className="h-auto p-4 flex flex-col items-center space-y-2" asChild>
              <Link href="/dashboard/settings">
                <Users className="w-6 h-6" />
                <span className="text-sm font-medium">Invite Team</span>
                <span className="text-xs text-muted-foreground">Add collaborators</span>
              </Link>
            </Button>

            <Button variant="outline" className="h-auto p-4 flex flex-col items-center space-y-2" asChild>
              <Link href="/dashboard/overview">
                <Activity className="w-6 h-6" />
                <span className="text-sm font-medium">View Analytics</span>
                <span className="text-xs text-muted-foreground">Team insights</span>
              </Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}