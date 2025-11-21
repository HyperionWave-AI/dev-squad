# How to Handle WebSocket Connections

**Collection:** howto
**Tags:** websocket, real-time, streaming, chat, go
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide explains how to implement WebSocket connections for real-time bidirectional communication. You'll learn connection lifecycle management, message handling, broadcasting, and graceful disconnection patterns using Gorilla WebSocket.

## Prerequisites

- Understanding of WebSocket protocol
- Go 1.25 with Gorilla WebSocket library
- Knowledge of goroutines and channels
- Familiarity with [Event Systems](../event-systems/websocket-streaming.md)

## When to Use This Guide

- Building chat/messaging features
- Implementing real-time updates (dashboards, notifications)
- Streaming AI responses
- Live collaboration features

---

## Steps

### Step 1: Install WebSocket Library

```bash
go get github.com/gorilla/websocket
```

### Step 2: Create WebSocket Upgrader

Configure HTTP-to-WebSocket connection upgrade:

```go
package handlers

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "go.uber.org/zap"
)

type ChatHandler struct {
    upgrader websocket.Upgrader
    logger   *zap.Logger
}

func NewChatHandler(logger *zap.Logger) *ChatHandler {
    return &ChatHandler{
        upgrader: websocket.Upgrader{
            // Allow connections from any origin (adjust for production)
            CheckOrigin: func(r *http.Request) bool {
                return true // TODO: Validate origin in production
            },
            // Buffer sizes
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
        },
        logger: logger,
    }
}
```

### Step 3: Implement Connection Handler

Upgrade HTTP connection and manage WebSocket lifecycle:

```go
func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
    // Extract user identity from JWT middleware
    userID, exists := c.Get("userId")
    if !exists {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }
    
    // Upgrade HTTP connection to WebSocket
    conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        h.logger.Error("Failed to upgrade connection",
            zap.Error(err),
            zap.String("userId", userID.(string)),
        )
        return
    }
    
    // Create client session
    client := &Client{
        ID:     generateClientID(),
        UserID: userID.(string),
        Conn:   conn,
        Send:   make(chan []byte, 256),
    }
    
    h.logger.Info("WebSocket connection established",
        zap.String("clientId", client.ID),
        zap.String("userId", client.UserID),
    )
    
    // Start goroutines for reading and writing
    go client.ReadPump(h)
    go client.WritePump(h)
}

type Client struct {
    ID     string
    UserID string
    Conn   *websocket.Conn
    Send   chan []byte
}
```

### Step 4: Implement Read Pump

Handle incoming messages from client:

```go
const (
    // Time allowed to read the next pong message from the peer
    pongWait = 60 * time.Second
    
    // Maximum message size allowed from peer
    maxMessageSize = 512 * 1024 // 512 KB
)

func (c *Client) ReadPump(h *ChatHandler) {
    defer func() {
        c.Conn.Close()
        h.logger.Info("Client disconnected", zap.String("clientId", c.ID))
    }()
    
    // Configure connection
    c.Conn.SetReadDeadline(time.Now().Add(pongWait))
    c.Conn.SetReadLimit(maxMessageSize)
    c.Conn.SetPongHandler(func(string) error {
        c.Conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    
    // Read messages loop
    for {
        _, message, err := c.Conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, 
                websocket.CloseGoingAway, 
                websocket.CloseAbnormalClosure) {
                h.logger.Warn("Unexpected close error",
                    zap.Error(err),
                    zap.String("clientId", c.ID),
                )
            }
            break
        }
        
        // Process incoming message
        h.HandleIncomingMessage(c, message)
    }
}

func (h *ChatHandler) HandleIncomingMessage(client *Client, raw []byte) {
    var msg IncomingMessage
    if err := json.Unmarshal(raw, &msg); err != nil {
        h.logger.Error("Failed to parse message",
            zap.Error(err),
            zap.String("clientId", client.ID),
        )
        return
    }
    
    // Route message based on type
    switch msg.Type {
    case "chat_message":
        h.handleChatMessage(client, &msg)
    case "ping":
        h.sendToClient(client, OutgoingMessage{Type: "pong"})
    default:
        h.logger.Warn("Unknown message type",
            zap.String("type", msg.Type),
            zap.String("clientId", client.ID),
        )
    }
}

type IncomingMessage struct {
    Type      string                 `json:"type"`
    Content   string                 `json:"content"`
    SessionID string                 `json:"sessionId"`
    Metadata  map[string]interface{} `json:"metadata"`
}
```

### Step 5: Implement Write Pump

Send messages to client with keep-alive:

```go
const (
    // Time allowed to write a message to the peer
    writeWait = 10 * time.Second
    
    // Send pings to peer with this period (must be less than pongWait)
    pingPeriod = (pongWait * 9) / 10 // 54 seconds
)

func (c *Client) WritePump(h *ChatHandler) {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.Conn.Close()
    }()
    
    for {
        select {
        case message, ok := <-c.Send:
            c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
            
            if !ok {
                // Channel closed
                c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            // Write message
            w, err := c.Conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)
            
            // Add queued messages to current message
            n := len(c.Send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.Send)
            }
            
            if err := w.Close(); err != nil {
                return
            }
            
        case <-ticker.C:
            // Send ping to keep connection alive
            c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

### Step 6: Implement Broadcasting

Manage multiple clients and broadcast messages:

```go
type Hub struct {
    // Registered clients
    clients map[*Client]bool
    
    // Inbound messages from clients
    broadcast chan []byte
    
    // Register requests from clients
    register chan *Client
    
    // Unregister requests from clients
    unregister chan *Client
    
    mu sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        broadcast:  make(chan []byte, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.Send)
            }
            h.mu.Unlock()
            
        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.Send <- message:
                default:
                    // Client buffer full, disconnect
                    close(client.Send)
                    delete(h.clients, client)
                }
            }
            h.mu.RUnlock()
        }
    }
}

// BroadcastToAll sends message to all connected clients
func (h *Hub) BroadcastToAll(message []byte) {
    h.broadcast <- message
}

// BroadcastToUser sends message to specific user's clients
func (h *Hub) BroadcastToUser(userID string, message []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    for client := range h.clients {
        if client.UserID == userID {
            select {
            case client.Send <- message:
            default:
                // Skip if buffer full
            }
        }
    }
}
```

### Step 7: Implement Streaming Response

Stream AI responses token-by-token:

```go
func (h *ChatHandler) handleChatMessage(client *Client, msg *IncomingMessage) {
    // Call AI service with streaming
    stream, err := h.aiService.ChatStream(context.Background(), msg.Content)
    if err != nil {
        h.sendError(client, "Failed to process message")
        return
    }
    
    // Stream tokens back to client
    for token := range stream {
        response := OutgoingMessage{
            Type:  "stream_token",
            Token: token.Text,
            Done:  token.Done,
        }
        
        data, _ := json.Marshal(response)
        select {
        case client.Send <- data:
        default:
            h.logger.Warn("Client send buffer full", zap.String("clientId", client.ID))
            return
        }
    }
}

type OutgoingMessage struct {
    Type    string `json:"type"`
    Token   string `json:"token,omitempty"`
    Content string `json:"content,omitempty"`
    Done    bool   `json:"done,omitempty"`
    Error   string `json:"error,omitempty"`
}
```

### Step 8: Register WebSocket Route

Add WebSocket endpoint to router:

```go
func RegisterRoutes(router *gin.Engine, chatHandler *ChatHandler) {
    // WebSocket endpoint
    router.GET("/ws/chat", 
        middleware.JWTAuthMiddleware(logger),
        chatHandler.HandleWebSocket,
    )
}
```

---

## Best Practices

### 1. Ping/Pong Keep-Alive
Always implement ping/pong to detect disconnections:
```go
pingPeriod < pongWait // pingPeriod must be less than pongWait
```

### 2. Message Size Limits
Prevent memory exhaustion with size limits:
```go
conn.SetReadLimit(512 * 1024) // 512 KB max
```

### 3. Graceful Shutdown
Close connections cleanly:
```go
defer conn.Close()
```

### 4. Origin Validation
Validate origins in production:
```go
CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    return origin == "https://yourdomain.com"
}
```

### 5. Buffered Channels
Use buffered channels to prevent blocking:
```go
Send: make(chan []byte, 256)
```

---

## Common Pitfalls

### 1. Blocking on Send
```go
// ❌ BAD - Can deadlock
client.Send <- message

// ✅ GOOD - Non-blocking send
select {
case client.Send <- message:
default:
    // Drop message or handle overflow
}
```

### 2. Not Handling Close Errors
```go
if websocket.IsUnexpectedCloseError(err, 
    websocket.CloseGoingAway, 
    websocket.CloseAbnormalClosure) {
    // Log unexpected closes
}
```

### 3. Missing Timeouts
Always set read/write deadlines to detect stuck connections.

---

## Related Documentation

- [WebSocket Streaming](../event-systems/websocket-streaming.md) - Architecture
- [JWT Authentication](./jwt-authentication-middleware.md) - Auth for WebSocket

---

## Troubleshooting

### Issue: "Connection closed unexpectedly"

**Solution:**
- Implement ping/pong keep-alive
- Check firewall/proxy timeout settings
- Increase `pongWait` timeout

### Issue: "Origin not allowed"

**Solution:**
```go
CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    allowed := []string{"http://localhost:3000", "https://app.com"}
    for _, a := range allowed {
        if origin == a {
            return true
        }
    }
    return false
}
```
