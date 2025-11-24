/**
 * WebSocket Connection State Machine
 * 
 * Explicit state machine for WebSocket connection management.
 * Defines all valid states and transitions.
 * 
 * STATE DIAGRAM:
 * 
 *                    ┌─────────────────────────────────────┐
 *                    │                                     │
 *                    ▼                                     │
 *            ┌──────────────────┐                         │
 *            │  DISCONNECTED    │◄────────────────────────┘
 *            └────────┬─────────┘
 *                     │ connect()
 *                     ▼
 *            ┌──────────────────┐
 *            │   CONNECTING     │
 *            └────────┬─────────┘
 *                     │
 *        ┌────────────┼────────────┐
 *        │            │            │
 *    success      timeout      error
 *        │            │            │
 *        ▼            ▼            ▼
 *    ┌──────────┐ ┌──────────┐ ┌──────────┐
 *    │CONNECTED │ │  ERROR   │ │  ERROR   │
 *    └────┬─────┘ └────┬─────┘ └────┬─────┘
 *         │            │            │
 *    disconnect()   reconnect()  reconnect()
 *         │            │            │
 *         └────────────┼────────────┘
 *                      ▼
 *            ┌──────────────────┐
 *            │ DISCONNECTING    │
 *            └────────┬─────────┘
 *                     │
 *                     ▼
 *            ┌──────────────────┐
 *            │  DISCONNECTED    │
 *            └──────────────────┘
 * 
 * STATES:
 * 
 * DISCONNECTED
 *   - No active connection
 *   - Can transition to: CONNECTING
 *   - Actions: connect()
 * 
 * CONNECTING
 *   - Connection attempt in progress
 *   - Can transition to: CONNECTED, ERROR, DISCONNECTED
 *   - Timeout: 10 seconds
 *   - Actions: None (wait for result)
 * 
 * CONNECTED
 *   - Active connection, ready to send/receive
 *   - Can transition to: DISCONNECTING, ERROR
 *   - Actions: sendMessage(), disconnect()
 * 
 * DISCONNECTING
 *   - Graceful disconnect in progress
 *   - Can transition to: DISCONNECTED
 *   - Actions: None (wait for completion)
 * 
 * ERROR
 *   - Connection error occurred
 *   - Can transition to: CONNECTING (retry), DISCONNECTED
 *   - Actions: reconnect(), disconnect()
 * 
 * TRANSITIONS:
 * 
 * DISCONNECTED → CONNECTING
 *   - Triggered by: connect() call
 *   - Action: Establish WebSocket connection
 * 
 * CONNECTING → CONNECTED
 *   - Triggered by: WebSocket onopen event
 *   - Action: Flush message queue, notify listeners
 * 
 * CONNECTING → ERROR
 *   - Triggered by: WebSocket onerror or timeout
 *   - Action: Notify listeners, schedule reconnect
 * 
 * CONNECTED → DISCONNECTING
 *   - Triggered by: disconnect() call
 *   - Action: Close WebSocket gracefully
 * 
 * CONNECTED → ERROR
 *   - Triggered by: WebSocket onerror
 *   - Action: Notify listeners, schedule reconnect
 * 
 * DISCONNECTING → DISCONNECTED
 *   - Triggered by: WebSocket onclose event
 *   - Action: Clean up resources
 * 
 * ERROR → CONNECTING
 *   - Triggered by: Automatic reconnect (exponential backoff)
 *   - Action: Attempt to reconnect
 * 
 * ERROR → DISCONNECTED
 *   - Triggered by: disconnect() call
 *   - Action: Cancel reconnect attempts, clean up
 * 
 * RECONNECTION STRATEGY:
 * 
 * - Exponential backoff: delay = min(1000 * 2^attempt, 30000)
 * - Jitter: Add random 0-10% to prevent thundering herd
 * - Max attempts: 5
 * - After max attempts: Stay in ERROR state, require manual reconnect
 * 
 * MESSAGE QUEUE:
 * 
 * - Messages sent while CONNECTING are queued
 * - Messages sent while DISCONNECTED are rejected
 * - Queue is flushed when transitioning to CONNECTED
 * - Queue is cleared when transitioning to DISCONNECTED
 * - Max queue size: 100 messages (prevents memory leaks)
 * 
 * ATOMIC OPERATIONS:
 * 
 * - State transitions are atomic (no race conditions)
 * - Message queue operations are atomic
 * - Listeners are notified after state change completes
 */

export const ConnectionStateMachine = {
  states: {
    DISCONNECTED: 'disconnected',
    CONNECTING: 'connecting',
    CONNECTED: 'connected',
    DISCONNECTING: 'disconnecting',
    ERROR: 'error',
  },

  transitions: {
    // From DISCONNECTED
    DISCONNECTED_TO_CONNECTING: {
      from: 'disconnected',
      to: 'connecting',
      trigger: 'connect',
    },

    // From CONNECTING
    CONNECTING_TO_CONNECTED: {
      from: 'connecting',
      to: 'connected',
      trigger: 'onopen',
    },
    CONNECTING_TO_ERROR: {
      from: 'connecting',
      to: 'error',
      trigger: 'onerror|timeout',
    },

    // From CONNECTED
    CONNECTED_TO_DISCONNECTING: {
      from: 'connected',
      to: 'disconnecting',
      trigger: 'disconnect',
    },
    CONNECTED_TO_ERROR: {
      from: 'connected',
      to: 'error',
      trigger: 'onerror',
    },

    // From DISCONNECTING
    DISCONNECTING_TO_DISCONNECTED: {
      from: 'disconnecting',
      to: 'disconnected',
      trigger: 'onclose',
    },

    // From ERROR
    ERROR_TO_CONNECTING: {
      from: 'error',
      to: 'connecting',
      trigger: 'reconnect',
    },
    ERROR_TO_DISCONNECTED: {
      from: 'error',
      to: 'disconnected',
      trigger: 'disconnect',
    },
  },

  validTransitions: {
    disconnected: ['connecting'],
    connecting: ['connected', 'error', 'disconnected'],
    connected: ['disconnecting', 'error'],
    disconnecting: ['disconnected'],
    error: ['connecting', 'disconnected'],
  },

  isValidTransition(from: string, to: string): boolean {
    const validTargets = this.validTransitions[from as keyof typeof this.validTransitions];
    return validTargets ? validTargets.includes(to) : false;
  },
};
