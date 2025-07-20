// app/dashboard/[projectId]/branches/page.tsx
"use client";

import { use, useState, useEffect } from "react";
import { GitBranch, Plus, Trash2, GitMerge, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ClientSideSuspense, LiveblocksProvider, RoomProvider } from "@liveblocks/react/suspense";

interface PageProps {
  params: Promise<{ projectId: string }>;
}

interface Branch {
  name: string;
  last_commit: string;
  last_commit_message: string;
  last_commit_author: string;
  last_commit_date: string;
  is_default: boolean;
  ahead_count: number;
  behind_count: number;
}

function BranchesContent({ projectId }: { projectId: string }) {
  const [branches, setBranches] = useState<Branch[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newBranchName, setNewBranchName] = useState('');
  const [creating, setCreating] = useState(false);

  const fetchBranches = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/v1/branches/${projectId}`);
      
      if (response.status === 404) {
        setError('Branches API not implemented yet. Backend endpoint needed: GET /api/v1/branches/:project');
        // Show mock data
        setBranches([
          {
            name: 'main',
            last_commit: 'abc123d',
            last_commit_message: 'Initial commit',
            last_commit_author: 'Developer',
            last_commit_date: new Date().toISOString(),
            is_default: true,
            ahead_count: 0,
            behind_count: 0
          }
        ]);
        return;
      }

      if (!response.ok) throw new Error('Failed to fetch branches');
      
      const data = await response.json();
      setBranches(data.branches || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load branches');
    } finally {
      setLoading(false);
    }
  };

  const createBranch = async () => {
    try {
      setCreating(true);
      const response = await fetch(`/api/v1/branches/${projectId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ 
          name: newBranchName,
          source: 'main' // Default source branch
        })
      });

      if (response.status === 404) {
        setError('Branch creation API not implemented yet. Backend endpoint needed: POST /api/v1/branches/:project');
        return;
      }

      if (!response.ok) throw new Error('Failed to create branch');

      await fetchBranches();
      setCreateDialogOpen(false);
      setNewBranchName('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create branch');
    } finally {
      setCreating(false);
    }
  };

  const deleteBranch = async (branchName: string) => {
    if (branchName === 'main') return; // Prevent deleting main branch
    
    try {
      const response = await fetch(`/api/v1/branches/${projectId}/${branchName}`, {
        method: 'DELETE'
      });

      if (response.status === 404) {
        setError('Branch deletion API not implemented yet. Backend endpoint needed: DELETE /api/v1/branches/:project/:branch');
        return;
      }

      if (!response.ok) throw new Error('Failed to delete branch');

      await fetchBranches();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete branch');
    }
  };

  useEffect(() => {
    fetchBranches();
  }, [projectId]);

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  return (
    <div className="h-full p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Branches</h2>
          <p className="text-muted-foreground">
            Manage project branches and merging
          </p>
        </div>
        
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="w-4 h-4 mr-2" />
              New Branch
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create New Branch</DialogTitle>
              <DialogDescription>
                Create a new branch from the main branch
              </DialogDescription>
            </DialogHeader>
            
            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium">Branch Name</label>
                <Input
                  placeholder="feature/new-feature"
                  value={newBranchName}
                  onChange={(e) => setNewBranchName(e.target.value)}
                />
              </div>
            </div>

            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setCreateDialogOpen(false)}
              >
                Cancel
              </Button>
              <Button
                onClick={createBranch}
                disabled={!newBranchName || creating}
              >
                {creating ? 'Creating...' : 'Create Branch'}
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

      {/* Branches List */}
      <div className="space-y-4">
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <div className="text-muted-foreground">Loading branches...</div>
          </div>
        ) : (
          branches.map((branch) => (
            <Card key={branch.name}>
              <CardContent className="p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <GitBranch className="w-5 h-5 text-muted-foreground" />
                    <div>
                      <div className="flex items-center gap-2">
                        <h3 className="font-medium">{branch.name}</h3>
                        {branch.is_default && (
                          <Badge variant="default">Default</Badge>
                        )}
                        {branch.ahead_count > 0 && (
                          <Badge variant="outline" className="text-green-600">
                            +{branch.ahead_count} ahead
                          </Badge>
                        )}
                        {branch.behind_count > 0 && (
                          <Badge variant="outline" className="text-red-600">
                            -{branch.behind_count} behind
                          </Badge>
                        )}
                      </div>
                      <div className="text-sm text-muted-foreground">
                        Last commit by {branch.last_commit_author} • {formatDate(branch.last_commit_date)}
                      </div>
                      <div className="text-xs font-mono text-muted-foreground">
                        {branch.last_commit}: {branch.last_commit_message}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    {!branch.is_default && branch.ahead_count > 0 && (
                      <Button size="sm" variant="outline">
                        <GitMerge className="w-4 h-4 mr-2" />
                        Merge
                      </Button>
                    )}
                    {!branch.is_default && (
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => deleteBranch(branch.name)}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))
        )}
      </div>
    </div>
  );
}

function BranchesPage({ params }: PageProps) {
  const { projectId } = use(params);

  return (
    <LiveblocksProvider authEndpoint="/api/liveblocks-auth">
      <RoomProvider id={`project-${projectId}`}>
        <ClientSideSuspense fallback={<BranchesLoader />}>
          <BranchesContent projectId={projectId} />
        </ClientSideSuspense>
      </RoomProvider>
    </LiveblocksProvider>
  );1
}

function BranchesLoader() {
  return (
    <div className="h-full flex items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
        <p className="text-muted-foreground">Loading branches...</p>
      </div>
    </div>
  );
}

export default BranchesPage;