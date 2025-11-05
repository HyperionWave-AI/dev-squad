import React, { useState } from 'react';
import { Send, Code as CodeIcon } from 'lucide-react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';
import * as Select from '@radix-ui/react-select';
import { Button } from '@/components/atoms/Button';
import { Input } from '@/components/atoms/Input';
import { Textarea } from '@/components/atoms/Textarea';
import { Badge } from '@/components/atoms/Badge';
import { PageHeader } from '@/components/organisms/PageHeader';

export function HTTPToolsPage() {
  const [method, setMethod] = useState('GET');
  const [url, setUrl] = useState('');
  const [body, setBody] = useState('');
  const [response, setResponse] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const handleSend = async () => {
    try {
      setLoading(true);
      const options: RequestInit = { method };
      if (method !== 'GET' && body) {
        options.body = body;
        options.headers = { 'Content-Type': 'application/json' };
      }
      const res = await fetch(url, options);
      const data = await res.text();
      setResponse({
        status: res.status,
        statusText: res.statusText,
        headers: Object.fromEntries(res.headers.entries()),
        body: data,
      });
    } catch (error: any) {
      setResponse({ error: error.message });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 via-white to-gray-50 dark:from-gray-950 dark:via-gray-900 dark:to-gray-950">
      <div className="container mx-auto p-6 space-y-6 max-w-7xl">
        {/* Header */}
        <PageHeader
          title="HTTP Tools"
          description="Test HTTP APIs and inspect responses"
          icon={<Send className="h-8 w-8" />}
          gradientFrom="#10b981"
          gradientTo="#059669"
        />

        {/* Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Request Builder - Glassmorphic Container */}
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg p-6 shadow-lg space-y-4">
            <h3 className="font-semibold text-gray-900 dark:text-gray-100">Request Builder</h3>
            <div className="flex gap-2">
              <Select.Root value={method} onValueChange={setMethod}>
                <Select.Trigger className="px-3 py-2 border border-border rounded-md bg-background">
                  <Select.Value />
                </Select.Trigger>
                <Select.Portal>
                  <Select.Content className="bg-background border border-border rounded-md shadow-lg z-50">
                    <Select.Viewport className="p-1">
                      {['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].map((m) => (
                        <Select.Item key={m} value={m} className="px-3 py-2 hover:bg-accent cursor-pointer rounded">
                          <Select.ItemText>{m}</Select.ItemText>
                        </Select.Item>
                      ))}
                    </Select.Viewport>
                  </Select.Content>
                </Select.Portal>
              </Select.Root>
              <Input
                placeholder="https://api.example.com/endpoint"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                className="flex-1"
              />
            </div>
            {method !== 'GET' && (
              <div>
                <label className="text-sm font-medium text-gray-900 dark:text-gray-100">Request Body (JSON)</label>
                <Textarea
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  placeholder='{"key": "value"}'
                  rows={10}
                  className="font-mono text-sm"
                />
              </div>
            )}
            <Button onClick={handleSend} disabled={loading || !url} className="w-full">
              {loading ? 'Sending...' : 'Send Request'}
            </Button>
          </div>

          {/* Response - Glassmorphic Container */}
          <div className="backdrop-blur-md bg-white/70 dark:bg-gray-800/70 border border-white/30 dark:border-gray-700/30 rounded-lg overflow-hidden shadow-lg">
            <div className="bg-white/30 dark:bg-gray-700/30 px-4 py-2 border-b border-white/30 dark:border-gray-700/30">
              <h3 className="font-semibold text-gray-900 dark:text-gray-100">Response</h3>
            </div>
            {!response ? (
              <div className="p-12 text-center">
                <CodeIcon className="h-12 w-12 mx-auto mb-4 opacity-50 text-gray-400 dark:text-gray-500" />
                <p className="text-gray-600 dark:text-gray-400">Send a request to see the response</p>
              </div>
            ) : response.error ? (
              <div className="p-4 text-red-500 dark:text-red-400">Error: {response.error}</div>
            ) : (
              <div className="p-4 space-y-3">
                <div className="flex gap-2 items-center">
                  <Badge variant={response.status < 400 ? 'success' : 'default'}>
                    {response.status} {response.statusText}
                  </Badge>
                </div>
                <div>
                  <h4 className="text-sm font-semibold mb-2 text-gray-900 dark:text-gray-100">Body</h4>
                  <SyntaxHighlighter
                    language="json"
                    style={vscDarkPlus}
                    customStyle={{ fontSize: '0.75rem', maxHeight: '400px' }}
                  >
                    {response.body}
                  </SyntaxHighlighter>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
