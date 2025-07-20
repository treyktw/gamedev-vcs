// app/dashboard/[projectId]/settings/page.tsx
"use client";

import { use, useState, useEffect } from "react";
import { Settings, Trash2, AlertTriangle, Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";

interface PageProps {
  params: Promise<{ projectId: string }>;
}

interface ProjectSettings {
  name: string;
  description: string;
  is_private: boolean;
  default_branch: string;
  allow_force_push: boolean;
  require_lock_for_edit: boolean;
}

function ProjectSettingsPage({ params }: PageProps) {
  const { projectId } = use(params);
  const [settings, setSettings] = useState<ProjectSettings>({
    name: '',
    description: '',
    is_private: true,
    default_branch: 'main',
    allow_force_push: false,
    require_lock_for_edit: true
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSettings = async () => {
    try {
      setLoading(true);
      const response = await fetch(`/api/v1/projects/${projectId}/settings`);
      
      if (response.status === 404) {
        setError('Project settings API not implemented yet. Backend endpoint needed: GET /api/v1/projects/:id/settings');
        // Use default settings
        setSettings({
          name: `Project ${projectId}`,
          description: 'Game development project',
          is_private: true,
          default_branch: 'main',
          allow_force_push: false,
          require_lock_for_edit: true
        });
        return;
      }

      if (!response.ok) throw new Error('Failed to fetch settings');
      
      const data = await response.json();
      setSettings(data.settings);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  const saveSettings = async () => {
    try {
      setSaving(true);
      const response = await fetch(`/api/v1/projects/${projectId}/settings`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings)
      });

      if (response.status === 404) {
        setError('Project settings update API not implemented yet. Backend endpoint needed: PUT /api/v1/projects/:id/settings');
        return;
      }

      if (!response.ok) throw new Error('Failed to save settings');
      
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
    } finally {
      setSaving(false);
    }
  };

  const deleteProject = async () => {
    if (!confirm('Are you sure you want to delete this project? This action cannot be undone.')) {
      return;
    }

    try {
      const response = await fetch(`/api/v1/projects/${projectId}`, {
        method: 'DELETE'
      });

      if (response.status === 404) {
        setError('Project deletion API not implemented yet. Backend endpoint needed: DELETE /api/v1/projects/:id');
        return;
      }

      if (!response.ok) throw new Error('Failed to delete project');
      
      // Redirect to dashboard
      window.location.href = '/dashboard';
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete project');
    }
  };

  useEffect(() => {
    fetchSettings();
  }, [projectId]);

  if (loading) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-muted-foreground">Loading settings...</div>
      </div>
    );
  }

  return (
    <div className="h-full p-6 space-y-6">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-bold">Project Settings</h2>
        <p className="text-muted-foreground">
          Configure your project settings and preferences
        </p>
      </div>

      {/* Error Display */}
      {error && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* General Settings */}
      <Card>
        <CardHeader>
          <CardTitle>General</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <label className="text-sm font-medium">Project Name</label>
            <Input
              value={settings.name}
              onChange={(e) => setSettings(prev => ({ ...prev, name: e.target.value }))}
              placeholder="Enter project name"
            />
          </div>
          
          <div>
            <label className="text-sm font-medium">Description</label>
            <Textarea
              value={settings.description}
              onChange={(e) => setSettings(prev => ({ ...prev, description: e.target.value }))}
              placeholder="Describe your project"
              rows={3}
            />
          </div>

          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium">Private Project</label>
              <p className="text-xs text-muted-foreground">
                Only team members can access this project
              </p>
            </div>
            <Switch
              checked={settings.is_private}
              onCheckedChange={(checked) => setSettings(prev => ({ ...prev, is_private: checked }))}
            />
          </div>
        </CardContent>
      </Card>

      {/* Collaboration Settings */}
      <Card>
        <CardHeader>
          <CardTitle>Collaboration</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium">Require File Locks</label>
              <p className="text-xs text-muted-foreground">
                Users must lock files before editing
              </p>
            </div>
            <Switch
              checked={settings.require_lock_for_edit}
              onCheckedChange={(checked) => setSettings(prev => ({ ...prev, require_lock_for_edit: checked }))}
            />
          </div>

          <div className="flex items-center justify-between">
            <div>
              <label className="text-sm font-medium">Allow Force Push</label>
              <p className="text-xs text-muted-foreground">
                Allow overwriting remote history
              </p>
            </div>
            <Switch
              checked={settings.allow_force_push}
              onCheckedChange={(checked) => setSettings(prev => ({ ...prev, allow_force_push: checked }))}
            />
          </div>
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex items-center justify-between">
        <Button onClick={saveSettings} disabled={saving}>
          <Save className="w-4 h-4 mr-2" />
          {saving ? 'Saving...' : 'Save Changes'}
        </Button>
      </div>

      <Separator />

      {/* Danger Zone */}
      <Card className="border-red-200">
        <CardHeader>
          <CardTitle className="text-red-600">Danger Zone</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between">
            <div>
              <h4 className="font-medium">Delete Project</h4>
              <p className="text-sm text-muted-foreground">
                Permanently delete this project and all its data
              </p>
            </div>
            <Button variant="destructive" onClick={deleteProject}>
              <Trash2 className="w-4 h-4 mr-2" />
              Delete Project
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default ProjectSettingsPage;