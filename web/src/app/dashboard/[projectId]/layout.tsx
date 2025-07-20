"use client";

import { use, useState, useEffect } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  FileText,
  GitBranch,
  Users,
  Settings,
  Activity,
  Search,
  Bell,
  GitCommit,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { useSession } from "next-auth/react";

interface LayoutProps {
  children: React.ReactNode;
  params: Promise<{ projectId: string }>;
}

interface Project {
  id: string;
  name: string;
  description?: string;
  is_private: boolean;
  default_branch: string;
  created_at: string;
  updated_at: string;
}

interface Notification {
  id: string;
  type: "file_locked" | "file_unlocked" | "member_joined" | "commit_created";
  message: string;
  timestamp: string;
  read: boolean;
}

export default function ProjectLayout({ children, params }: LayoutProps) {
  const { projectId } = use(params);
  const pathname = usePathname();
  const [project, setProject] = useState<Project | null>(null);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);

  const { data: session } = useSession();

  // Navigation items
  const navigation = [
    {
      name: "Files",
      href: `/dashboard/${projectId}/files`,
      icon: FileText,
      current: pathname.includes("/files"),
    },
    {
      name: "Commits",
      href: `/dashboard/${projectId}/commits`,
      icon: GitCommit,
      current: pathname.includes("/commits"),
    },
    {
      name: "Branches",
      href: `/dashboard/${projectId}/branches`,
      icon: GitBranch,
      current: pathname.includes("/branches"),
    },
    {
      name: "Team",
      href: `/dashboard/${projectId}/team`,
      icon: Users,
      current: pathname.includes("/team"),
    },
    {
      name: "Activity",
      href: `/dashboard/${projectId}/activity`,
      icon: Activity,
      current: pathname.includes("/activity"),
    },
    {
      name: "Settings",
      href: `/dashboard/${projectId}/settings`,
      icon: Settings,
      current: pathname.includes("/settings"),
    },
  ];

  // Fetch project details
  useEffect(() => {
    const fetchProject = async () => {
      try {
        const response = await fetch(`/api/v1/projects/${projectId}`);
        if (response.status === 404) {
          console.log("Project API not fully implemented. Using basic data.");
          setProject({
            id: projectId,
            name: `Project ${projectId}`,
            description: "Loading project details...",
            is_private: true,
            default_branch: "main",
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          });
          return;
        }

        if (!response.ok) throw new Error("Failed to fetch project");

        const data = await response.json();
        setProject(data.project);
      } catch (error) {
        console.error("Failed to fetch project:", error);
        // Fallback project data
        setProject({
          id: projectId,
          name: `Project ${projectId}`,
          description: "Project details unavailable",
          is_private: true,
          default_branch: "main",
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        });
      } finally {
        setLoading(false);
      }
    };

    fetchProject();

    // Set up notifications WebSocket
    const ws = new WebSocket(
      `ws://localhost:8080/api/v1/collaboration/ws?project_id=${projectId}&token=${session?.accessToken || ''}`
    );
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      const notification: Notification = {
        id: Date.now().toString(),
        type: data.type,
        message: data.message || `${data.type} event occurred`,
        timestamp: new Date().toISOString(),
        read: false,
      };
      setNotifications((prev) => [notification, ...prev.slice(0, 9)]); // Keep last 10
    };
    ws.onerror = () => {
      console.log("Notifications WebSocket not available");
    };

    return () => ws.close();
  }, [projectId]);

  const unreadCount = notifications.filter((n) => !n.read).length;

  if (loading) {
    return (
      <div className="h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    );
  }

  return (
    <div className="h-screen flex flex-col">
      {/* Top Navigation */}
      <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="flex h-14 items-center px-4">
          {/* Project Info */}
          <div className="flex items-center gap-4 mr-8">
            <div>
              <h1 className="font-semibold text-lg">{project?.name}</h1>
              <p className="text-xs text-muted-foreground">
                {project?.is_private ? "Private" : "Public"} •{" "}
                {project?.default_branch}
              </p>
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex items-center space-x-6 lg:space-x-8 mr-8">
            {navigation.map((item) => {
              const Icon = item.icon;
              return (
                <Link
                  key={item.name}
                  href={item.href}
                  className={cn(
                    "flex items-center gap-2 text-sm font-medium transition-colors hover:text-primary",
                    item.current
                      ? "text-foreground border-b-2 border-primary pb-3 mb-[-1px]"
                      : "text-muted-foreground"
                  )}
                >
                  <Icon className="w-4 h-4" />
                  {item.name}
                </Link>
              );
            })}
          </nav>

          {/* Right side */}
          <div className="ml-auto flex items-center gap-4">
            {/* Global Search */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground w-4 h-4" />
              <Input
                placeholder="Search files, commits, issues..."
                className="pl-10 w-80"
                onFocus={() => {
                  console.log(
                    "Global search not implemented yet. Backend endpoint needed: GET /api/v1/search?q=..."
                  );
                }}
              />
            </div>

            {/* Notifications */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="relative">
                  <Bell className="w-4 h-4" />
                  {unreadCount > 0 && (
                    <Badge
                      variant="destructive"
                      className="absolute -top-1 -right-1 h-5 w-5 flex items-center justify-center p-0 text-xs"
                    >
                      {unreadCount}
                    </Badge>
                  )}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-80">
                <div className="p-3 border-b">
                  <h4 className="font-medium">Notifications</h4>
                </div>
                {notifications.length > 0 ? (
                  notifications.slice(0, 5).map((notification) => (
                    <DropdownMenuItem
                      key={notification.id}
                      className="flex flex-col items-start p-3"
                      onClick={() => {
                        setNotifications((prev) =>
                          prev.map((n) =>
                            n.id === notification.id ? { ...n, read: true } : n
                          )
                        );
                      }}
                    >
                      <div className="flex items-center justify-between w-full">
                        <span className="font-medium text-sm">
                          {notification.type.replace("_", " ")}
                        </span>
                        {!notification.read && (
                          <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">
                        {notification.message}
                      </p>
                      <span className="text-xs text-muted-foreground mt-1">
                        {new Date(notification.timestamp).toLocaleTimeString()}
                      </span>
                    </DropdownMenuItem>
                  ))
                ) : (
                  <DropdownMenuItem className="text-center text-muted-foreground p-6">
                    No notifications
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>

            {/* User Menu */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  className="relative h-8 w-8 rounded-full"
                >
                  <Avatar className="h-8 w-8">
                    <AvatarImage src="" alt="User" />
                    <AvatarFallback>U</AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem>Profile</DropdownMenuItem>
                <DropdownMenuItem>Settings</DropdownMenuItem>
                <DropdownMenuItem>Sign out</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 overflow-hidden">{children}</main>
    </div>
  );
}
