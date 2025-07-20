'use client';

import { useState } from 'react';
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { apiClient } from "@/lib/api-client";
import { CheckCircle, XCircle, Loader2, Play } from "lucide-react";

interface TestResult {
  name: string;
  status: 'pending' | 'success' | 'error';
  message?: string;
  duration?: number;
}

export function ConnectionTest() {
  const [tests, setTests] = useState<TestResult[]>([
    { name: 'Health Check', status: 'pending' },
    { name: 'Storage Stats', status: 'pending' },
    { name: 'Authentication', status: 'pending' },
  ]);
  const [isRunning, setIsRunning] = useState(false);

  const updateTest = (index: number, update: Partial<TestResult>) => {
    setTests(prev => prev.map((test, i) => i === index ? { ...test, ...update } : test));
  };

  const runTests = async () => {
    setIsRunning(true);
    
    // Reset all tests
    setTests(prev => prev.map(test => ({ ...test, status: 'pending' as const, message: undefined, duration: undefined })));

    // Test 1: Health Check
    try {
      const start = Date.now();
      await apiClient.healthCheck();
      const duration = Date.now() - start;
      updateTest(0, { 
        status: 'success', 
        message: `Server is healthy (${duration}ms)`,
        duration 
      });
    } catch (error) {
      updateTest(0, { 
        status: 'error', 
        message: error instanceof Error ? error.message : 'Health check failed' 
      });
    }

    // Test 2: Storage Stats
    try {
      const start = Date.now();
      const result = await apiClient.getStorageStats();
      const duration = Date.now() - start;
      updateTest(1, { 
        status: 'success', 
        message: `${result.stats.totalFiles} files, ${(result.stats.totalSize / 1024 / 1024).toFixed(1)}MB (${duration}ms)`,
        duration 
      });
    } catch (error) {
      updateTest(1, { 
        status: 'error', 
        message: error instanceof Error ? error.message : 'Storage stats failed' 
      });
    }

    // Test 3: Authentication (try to get projects)
    try {
      const start = Date.now();
      const result = await apiClient.getProjects();
      const duration = Date.now() - start;
      updateTest(2, { 
        status: 'success', 
        message: `Found ${result.projects.length} projects (${duration}ms)`,
        duration 
      });
    } catch (error) {
      updateTest(2, { 
        status: 'error', 
        message: error instanceof Error ? error.message : 'Project fetch failed' 
      });
    }

    setIsRunning(false);
  };

  const getStatusIcon = (status: TestResult['status']) => {
    switch (status) {
      case 'success':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'error':
        return <XCircle className="w-4 h-4 text-red-500" />;
      case 'pending':
        return isRunning ? <Loader2 className="w-4 h-4 animate-spin" /> : <div className="w-4 h-4 rounded-full border border-muted-foreground" />;
    }
  };

  const getStatusBadge = (status: TestResult['status']) => {
    switch (status) {
      case 'success':
        return <Badge variant="default" className="bg-green-500">Pass</Badge>;
      case 'error':
        return <Badge variant="destructive">Fail</Badge>;
      case 'pending':
        return <Badge variant="secondary">Pending</Badge>;
    }
  };

  const allPassed = tests.every(test => test.status === 'success');
  const anyFailed = tests.some(test => test.status === 'error');

  return (
    <Card className="border-2">
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>API Connection Test</span>
          {allPassed && <Badge variant="default" className="bg-green-500">All Tests Passed</Badge>}
          {anyFailed && <Badge variant="destructive">Tests Failed</Badge>}
        </CardTitle>
        <CardDescription>
          Test connection to your VCS backend server at {process.env.NEXT_PUBLIC_VCS_API_URL}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-3">
          {tests.map((test, index) => (
            <div key={test.name} className="flex items-center justify-between p-3 border rounded-lg">
              <div className="flex items-center space-x-3">
                {getStatusIcon(test.status)}
                <div>
                  <p className="font-medium">{test.name}</p>
                  {test.message && (
                    <p className="text-sm text-muted-foreground">{test.message}</p>
                  )}
                </div>
              </div>
              {getStatusBadge(test.status)}
            </div>
          ))}
        </div>

        <Button 
          onClick={runTests} 
          disabled={isRunning}
          className="w-full"
        >
          {isRunning ? (
            <>
              <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              Running Tests...
            </>
          ) : (
            <>
              <Play className="w-4 h-4 mr-2" />
              Run Connection Tests
            </>
          )}
        </Button>

        {/* Quick Setup Guide */}
        <div className="mt-6 p-4 bg-muted/50 rounded-lg">
          <h4 className="font-medium mb-2">Quick Setup Guide:</h4>
          <ol className="text-sm text-muted-foreground space-y-1 list-decimal list-inside">
            <li>Make sure your Go VCS server is running on port 8080</li>
            <li>Check that Redis and ClickHouse containers are up</li>
            <li>Verify NEXT_PUBLIC_VCS_API_URL in your .env.local</li>
            <li>Ensure CORS is enabled in your Go server</li>
          </ol>
        </div>
      </CardContent>
    </Card>
  );
}