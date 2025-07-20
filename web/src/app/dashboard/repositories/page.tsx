import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { 
  DropdownMenu, 
  DropdownMenuContent, 
  DropdownMenuItem, 
  DropdownMenuTrigger 
} from "@/components/ui/dropdown-menu";
import { 
  Folder, 
  Search, 
  Filter, 
  Plus, 
  GitBranch,
  Users,
  Clock,
  Lock,
  Star,
  Archive,
  MoreHorizontal,
  Eye,
  Settings
} from "lucide-react";
import Link from "next/link";

const repositories = [
  {
    id: "1",
    name: "awesome-game",
    description: "Main UE5 game project with character systems and level design",
    visibility: "private",
    language: "Blueprint",
    stars: 0,
    forks: 0,
    size: "2.1 GB",
    contributors: 5,
    lastCommit: "2 hours ago",
    lastCommitMessage: "Updated character movement system",
    lastCommitAuthor: "Alice Johnson",
    isArchived: false,
    hasLocks: true,
    lockCount: 3,
    branch: "main"
  },
  {
    id: "2", 
    name: "character-assets",
    description: "3D character models, textures, and animation files",
    visibility: "private",
    language: "Assets",
    stars: 2,
    forks: 1,
    size: "890 MB",
    contributors: 3,
    lastCommit: "1 day ago",
    lastCommitMessage: "Added new hero character animations",
    lastCommitAuthor: "Bob Smith",
    isArchived: false,
    hasLocks: false,
    lockCount: 0,
    branch: "develop"
  },
  {
    id: "3",
    name: "level-designs", 
    description: "Environment assets and level layout files",
    visibility: "private",
    language: "UE5",
    stars: 1,
    forks: 0,
    size: "1.5 GB",
    contributors: 4,
    lastCommit: "3 days ago",
    lastCommitMessage: "Optimized lighting for forest level",
    lastCommitAuthor: "Charlie Brown",
    isArchived: false,
    hasLocks: true,
    lockCount: 1,
    branch: "main"
  },
  {
    id: "4",
    name: "old-prototype",
    description: "Legacy prototype from early development phase",
    visibility: "private", 
    language: "UE4",
    stars: 0,
    forks: 0,
    size: "450 MB",
    contributors: 2,
    lastCommit: "2 months ago",
    lastCommitMessage: "Final prototype commit",
    lastCommitAuthor: "Diana Prince",
    isArchived: true,
    hasLocks: false,
    lockCount: 0,
    branch: "main"
  }
];

function RepositoryCard({ repo }: { repo: typeof repositories[0] }) {
  return (
    <Card className="border-2 hover:shadow-md transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1 flex-1">
            <div className="flex items-center space-x-2">
              <Link 
                href={`/your-username/${repo.name}`}
                className="text-lg font-semibold text-primary hover:underline"
              >
                {repo.name}
              </Link>
              <Badge variant="outline" className="text-xs">
                {repo.visibility}
              </Badge>
              {repo.isArchived && (
                <Badge variant="secondary" className="text-xs">
                  <Archive className="w-3 h-3 mr-1" />
                  Archived
                </Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">
              {repo.description}
            </p>
          </div>
          
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreHorizontal className="w-4 h-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem asChild>
                <Link href={`/your-username/${repo.name}`}>
                  <Eye className="w-4 h-4 mr-2" />
                  View Repository
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <Link href={`/your-username/${repo.name}/settings`}>
                  <Settings className="w-4 h-4 mr-2" />
                  Settings
                </Link>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      
      <CardContent className="pt-0">
        <div className="flex items-center justify-between text-sm text-muted-foreground mb-3">
          <div className="flex items-center space-x-4">
            {repo.language && (
              <span className="flex items-center">
                <div className="w-3 h-3 rounded-full bg-primary mr-1" />
                {repo.language}
              </span>
            )}
            <span className="flex items-center">
              <Star className="w-3 h-3 mr-1" />
              {repo.stars}
            </span>
            <span className="flex items-center">
              <GitBranch className="w-3 h-3 mr-1" />
              {repo.forks}
            </span>
            <span className="flex items-center">
              <Users className="w-3 h-3 mr-1" />
              {repo.contributors}
            </span>
            <span>{repo.size}</span>
          </div>
          
          {repo.hasLocks && (
            <Badge variant="destructive" className="text-xs">
              <Lock className="w-3 h-3 mr-1" />
              {repo.lockCount} locked
            </Badge>
          )}
        </div>
        
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <div className="flex items-center space-x-1">
            <span>Updated {repo.lastCommit} by {repo.lastCommitAuthor}</span>
          </div>
          <Badge variant="outline" className="text-xs">
            {repo.branch}
          </Badge>
        </div>
      </CardContent>
    </Card>
  );
}

export default function RepositoriesPage() {
  const activeRepos = repositories.filter(repo => !repo.isArchived);
  const archivedRepos = repositories.filter(repo => repo.isArchived);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Repositories</h1>
          <p className="text-muted-foreground">
            Manage your game development projects and assets
          </p>
        </div>
        <Button asChild>
          <Link href="/dashboard/repositories/new">
            <Plus className="w-4 h-4 mr-2" />
            New Repository
          </Link>
        </Button>
      </div>

      {/* Search and Filters */}
      <div className="flex items-center space-x-4">
        <div className="flex-1 max-w-md">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <Input 
              placeholder="Search repositories..." 
              className="pl-10"
            />
          </div>
        </div>
        
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm">
              <Filter className="w-4 h-4 mr-2" />
              Filter
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-48">
            <DropdownMenuItem>All repositories</DropdownMenuItem>
            <DropdownMenuItem>Public</DropdownMenuItem>
            <DropdownMenuItem>Private</DropdownMenuItem>
            <DropdownMenuItem>Archived</DropdownMenuItem>
            <DropdownMenuItem>With locks</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Repository Tabs */}
      <Tabs defaultValue="active" className="space-y-6">
        <TabsList>
          <TabsTrigger value="active" className="flex items-center space-x-2">
            <Folder className="w-4 h-4" />
            <span>Active ({activeRepos.length})</span>
          </TabsTrigger>
          <TabsTrigger value="archived" className="flex items-center space-x-2">
            <Archive className="w-4 h-4" />
            <span>Archived ({archivedRepos.length})</span>
          </TabsTrigger>
        </TabsList>

        <TabsContent value="active" className="space-y-4">
          {activeRepos.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-1 lg:grid-cols-2">
              {activeRepos.map((repo) => (
                <RepositoryCard key={repo.id} repo={repo} />
              ))}
            </div>
          ) : (
            <Card className="border-2 border-dashed">
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Folder className="w-12 h-12 text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No repositories yet</h3>
                <p className="text-muted-foreground text-center mb-4 max-w-md">
                  Create your first repository to start managing your game development assets and collaborate with your team.
                </p>
                <Button asChild>
                  <Link href="/dashboard/repositories/new">
                    <Plus className="w-4 h-4 mr-2" />
                    Create Repository
                  </Link>
                </Button>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="archived" className="space-y-4">
          {archivedRepos.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-1 lg:grid-cols-2">
              {archivedRepos.map((repo) => (
                <RepositoryCard key={repo.id} repo={repo} />
              ))}
            </div>
          ) : (
            <Card className="border-2 border-dashed">
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Archive className="w-12 h-12 text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No archived repositories</h3>
                <p className="text-muted-foreground text-center max-w-md">
                  Archived repositories will appear here. Archive repositories that are no longer actively developed.
                </p>
              </CardContent>
            </Card>
          )}
        </TabsContent>
      </Tabs>

      {/* Quick Stats */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="border-2">
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Storage</p>
                <p className="text-2xl font-bold">4.9 GB</p>
              </div>
              <div className="text-right text-muted-foreground">
                <p className="text-xs">of 10 GB</p>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="border-2">
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Active Locks</p>
                <p className="text-2xl font-bold">4</p>
              </div>
              <Lock className="w-5 h-5 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>

        <Card className="border-2">
          <CardContent className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Team Members</p>
                <p className="text-2xl font-bold">7</p>
              </div>
              <Users className="w-5 h-5 text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}