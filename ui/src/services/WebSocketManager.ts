/**
 * WebSocketManager - Thread-safe WebSocket connection manager
 *
 * Fixes race conditions:
 * 1. State check-and-send race - Atomic operations with proper locking
 * 2. Cleanup race - Coordinated cleanup with state machine
 * 3. Connection state inconsistency - Single source of truth
 * 4. Message queue conflicts - Synchronized queue operations
 */

export const ConnectionState = {
  DISCONNECTED: 'DISCONNECTED',
  CONNECTING: 'CONNECTING',
  CONNECTED: 'CONNECTED',
  DISCONNECTING: 'DISCONNECTING',
  ERROR: 'ERROR'
} as const;

export type ConnectionState = typeof ConnectionState[keyof typeof ConnectionState];

interface QueuedMessage {
  content: string;
  timestamp: number;
  resolve: () => void;
  reject: (error: Error) => void;
}

export class WebSocketManager {
  private ws: WebSocket | null = null;
  private state: ConnectionState = ConnectionState.DISCONNECTED;
  private sessionId: string | null = null;
  private messageQueue: QueuedMessage[] = [];
  private callbacks: Map<string, Function> = new Map();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectAttempts: number = 0;
  private maxReconnectAttempts: number = 5;
  private readonly MAX_QUEUE_SIZE = 100;
  private readonly QUEUE_TIMEOUT = 30000; // 30 seconds

  // Lock for atomic state transitions
  private stateLock: Promise<void> = Promise.resolve();

  // Lock for message queue operations
  private queueLock: Promise<void> = Promise.resolve();

  /**
   * Connect to WebSocket with atomic state transition
   */
  async connect(sessionId: string, callbacks: Record<string, Function>): Promise<void> {
    return this.withStateLock(async () => {
      // Prevent concurrent connections
      if (this.state === ConnectionState.CONNECTING || this.state === ConnectionState.CONNECTED) {
        if (this.sessionId === sessionId) {
          console.log('[WSManager] Already connected to session:', sessionId);
          return;
        }
        // Different session - disconnect first
        await this.disconnect();
      }

      this.state = ConnectionState.CONNECTING;
      this.sessionId = sessionId;
      this.callbacks = new Map(Object.entries(callbacks));

      try {
        await this.establishConnection(sessionId);
      } catch (error) {
        this.state = ConnectionState.ERROR;
        throw error;
      }
    });
  }

  /**
   * Send message with atomic state check
   */
  async sendMessage(content: string): Promise<void> {
    return new Promise((resolve, reject) => {
      this.withQueueLock(async () => {
        // Check queue size before adding
        if (this.messageQueue.length >= this.MAX_QUEUE_SIZE) {
          console.warn('[WSManager] Message queue full (max 100), rejecting message');
          reject(new Error('Message queue full - connection unstable'));
          return;
        }

        if (this.state !== ConnectionState.CONNECTED) {
          // Queue message for later
          this.messageQueue.push({ content, timestamp: Date.now(), resolve, reject });
          console.log('[WSManager] Message queued (not connected):', content.substring(0, 50));

          // Reject after timeout if still not connected
          setTimeout(() => {
            const index = this.messageQueue.findIndex(m => m.content === content);
            if (index !== -1) {
              const msg = this.messageQueue.splice(index, 1)[0];
              console.warn('[WSManager] Message queue timeout after 30s, rejecting:', content.substring(0, 50));
              msg.reject(new Error('Message queue timeout - connection not established within 30 seconds'));
            }
          }, this.QUEUE_TIMEOUT); // 30 second timeout
          return;
        }

        // Send immediately
        try {
          this.sendImmediately(content);
          resolve();
        } catch (error) {
          reject(error instanceof Error ? error : new Error('Send failed'));
        }
      });
    });
  }

  /**
   * Disconnect with atomic cleanup
   */
  async disconnect(): Promise<void> {
    return this.withStateLock(async () => {
      if (this.state === ConnectionState.DISCONNECTED) {
        return;
      }

      // Cancel reconnection attempts
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
      }

      this.state = ConnectionState.DISCONNECTING;

      // Reject all queued messages
      await this.withQueueLock(async () => {
        this.messageQueue.forEach(msg => {
          msg.reject(new Error('Connection closed'));
        });
        this.messageQueue = [];
      });

      // Close WebSocket if exists
      if (this.ws) {
        try {
          // Check state atomically before closing
          if (this.ws.readyState === WebSocket.OPEN ||
              this.ws.readyState === WebSocket.CONNECTING) {
            this.ws.close();
          }
        } catch (error) {
          console.error('[WSManager] Error closing WebSocket:', error);
        }
        this.ws = null;
      }

      this.state = ConnectionState.DISCONNECTED;
      this.sessionId = null;
      this.reconnectAttempts = 0;
    });
  }

  /**
   * Cleanup all resources (for memory leak prevention)
   */
  async cleanup(): Promise<void> {
    await this.disconnect();
    this.callbacks.clear();
  }

  /**
   * Get current state (thread-safe read)
   */
  getState(): ConnectionState {
    return this.state;
  }

  /**
   * Get current session ID (thread-safe read)
   */
  getSessionId(): string | null {
    return this.sessionId;
  }

  /**
   * Check if connected (atomic check)
   */
  isConnected(): boolean {
    return this.state === ConnectionState.CONNECTED &&
           this.ws !== null &&
           this.ws.readyState === WebSocket.OPEN;
  }

  // Private helper methods

  private establishConnection(sessionId: string): Promise<void> {
    return new Promise((resolve, reject) => {
      const wsUrl = `${this.getWebSocketUrl()}/chat/stream?sessionId=${sessionId}`;
      const ws = new WebSocket(wsUrl);
      let resolved = false;

      ws.onopen = () => {
        if (resolved) return; // Prevent double-resolve
        resolved = true;

        this.ws = ws;
        this.state = ConnectionState.CONNECTED;
        this.reconnectAttempts = 0;

        console.log('[WSManager] Connected to session:', sessionId);
        this.callbacks.get('onOpen')?.();

        // Flush queued messages
        this.flushMessageQueue();

        resolve();
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.callbacks.get('onMessage')?.(data);
        } catch (error) {
          console.error('[WSManager] Message parse error:', error);
          this.callbacks.get('onError')?.(error);
        }
      };

      ws.onerror = (event) => {
        console.error('[WSManager] WebSocket error:', event);

        if (!resolved) {
          resolved = true;
          this.state = ConnectionState.ERROR;
          reject(new Error('WebSocket connection failed'));
        }

        this.callbacks.get('onError')?.(new Error('WebSocket error'));
      };

      ws.onclose = () => {
        console.log('[WSManager] WebSocket closed');

        if (!resolved) {
          resolved = true;
          reject(new Error('WebSocket closed before connection established'));
        }

        // Only attempt reconnect if we were connected (not manual disconnect)
        if (this.state === ConnectionState.CONNECTED) {
          this.attemptReconnect();
        }

        this.callbacks.get('onClose')?.();
      };

      // Connection timeout
      setTimeout(() => {
        if (!resolved) {
          resolved = true;
          this.state = ConnectionState.ERROR;
          ws.close();
          reject(new Error('Connection timeout'));
        }
      }, 10000); // 10 second timeout
    });
  }

  private sendImmediately(content: string): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket not connected');
    }

    try {
      this.ws.send(JSON.stringify({ content }));
      console.log('[WSManager] Message sent:', content.substring(0, 50));
    } catch (error) {
      console.error('[WSManager] Send error:', error);
      throw error;
    }
  }

  private async flushMessageQueue(): Promise<void> {
    await this.withQueueLock(async () => {
      if (this.messageQueue.length === 0) return;

      console.log(`[WSManager] Flushing ${this.messageQueue.length} queued messages`);

      const queue = [...this.messageQueue];
      this.messageQueue = [];

      for (const msg of queue) {
        try {
          this.sendImmediately(msg.content);
          msg.resolve();
        } catch (error) {
          msg.reject(error instanceof Error ? error : new Error('Send failed'));
        }
      }
    });
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[WSManager] Max reconnect attempts reached');
      this.state = ConnectionState.ERROR;
      return;
    }

    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;

    console.log(`[WSManager] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    this.reconnectTimer = setTimeout(() => {
      if (this.sessionId) {
        this.connect(this.sessionId, Object.fromEntries(this.callbacks))
          .catch(error => {
            console.error('[WSManager] Reconnect failed:', error);
          });
      }
    }, delay);
  }

  private async withStateLock<T>(fn: () => Promise<T>): Promise<T> {
    const previous = this.stateLock;
    let resolveLock: () => void;

    this.stateLock = new Promise(resolve => {
      resolveLock = resolve;
    });

    try {
      await previous;
      return await fn();
    } finally {
      resolveLock!();
    }
  }

  private async withQueueLock<T>(fn: () => Promise<T>): Promise<T> {
    const previous = this.queueLock;
    let resolveLock: () => void;

    this.queueLock = new Promise(resolve => {
      resolveLock = resolve;
    });

    try {
      await previous;
      return await fn();
    } finally {
      resolveLock!();
    }
  }

  private getWebSocketUrl(): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/api/v1`;
  }
}
