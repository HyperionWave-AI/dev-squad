# Hyperion Chat System - Metrics Setup Guide

## 🎯 Overview

This guide explains how to set up Prometheus and Grafana to monitor your chat system using the metrics we just implemented.

## 📊 What You Get

- **51 Total Metrics** exposed at `http://localhost:5555/metrics`
- **Real-time Dashboards** with 14 visualization panels
- **Production Observability** for performance, security, and health monitoring

---

## 🚀 Quick Setup (Docker Compose)

### 1. Create `docker-compose.yml`

```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: hyperion-prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--storage.tsdb.retention.time=30d'
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: hyperion-grafana
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana-provisioning:/etc/grafana/provisioning
    depends_on:
      - prometheus
    networks:
      - monitoring

volumes:
  prometheus-data:
  grafana-data:

networks:
  monitoring:
    driver: bridge
```

### 2. Create `prometheus.yml`

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'hyperion-dev'
    environment: 'development'

scrape_configs:
  - job_name: 'hyperion-chat'
    static_configs:
      - targets: ['host.docker.internal:5555']  # macOS/Windows
        # Use 'localhost:5555' on Linux
    metrics_path: '/metrics'
    scrape_interval: 10s
```

### 3. Create Grafana Provisioning Directory

```bash
mkdir -p grafana-provisioning/datasources
mkdir -p grafana-provisioning/dashboards
```

### 4. Create `grafana-provisioning/datasources/prometheus.yml`

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

### 5. Create `grafana-provisioning/dashboards/dashboards.yml`

```yaml
apiVersion: 1

providers:
  - name: 'Hyperion Dashboards'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
```

### 6. Copy Dashboard JSON

```bash
cp grafana-dashboard.json grafana-provisioning/dashboards/
```

### 7. Start Everything

```bash
# Start Prometheus and Grafana
docker-compose up -d

# Verify they're running
docker-compose ps

# Check Prometheus targets
open http://localhost:9090/targets

# Open Grafana
open http://localhost:3001
```

---

## 🔐 Access Details

### Prometheus
- **URL**: http://localhost:9090
- **Targets**: http://localhost:9090/targets (verify your app is being scraped)
- **Graph**: http://localhost:9090/graph

### Grafana
- **URL**: http://localhost:3001
- **Username**: `admin`
- **Password**: `admin` (change on first login)
- **Dashboard**: Look for "Hyperion Chat System - Production Metrics"

---

## 📈 Dashboard Overview

### Top Row - Key Metrics
1. **Active WebSocket Connections** - Current concurrent users
2. **Connection Rate** - New connections per minute
3. **Total Sessions Created** - Lifetime session count
4. **Validation Rejections** - Security monitoring (DoS detection)

### Row 2 - Throughput
5. **Message Throughput** - Messages sent/received per minute
6. **Active Connections Over Time** - Connection trend graph

### Row 3 - Security & Performance
7. **Validation Rejections by Layer** - Where messages are being blocked
8. **WebSocket Connection Duration** - P95/P99 latency

### Row 4 - AI Performance
9. **AI Streaming Token Rate** - Tokens generated per minute
10. **AI Streaming Duration** - P50/P95/P99 latency

### Row 5 - Database Performance
11. **Messages Saved by Role** - User vs assistant message rates
12. **Message Save Duration** - Database write latency

### Row 6 - System Health
13. **Go Runtime Goroutines** - Concurrency monitoring
14. **Go Runtime Memory** - Memory usage tracking

---

## 🎨 Example Queries (for Prometheus)

### Connection Metrics
```promql
# Current active connections
chat_websocket_active_connections

# Connection rate (per minute)
rate(chat_websocket_connections_total[5m]) * 60

# Average connection duration
rate(chat_websocket_connection_duration_seconds_sum[5m]) / rate(chat_websocket_connection_duration_seconds_count[5m])
```

### Message Metrics
```promql
# Message throughput (per second)
rate(chat_websocket_messages_sent_total[1m])
rate(chat_websocket_messages_received_total[1m])

# Total messages saved
sum(chat_messages_saved_total) by (role)
```

### Validation & Security
```promql
# Validation rejection rate
sum(rate(chat_message_validation_rejections_total[5m])) * 60

# Rejections by layer
sum(rate(chat_message_validation_rejections_total[5m])) by (layer)

# Size exceeded violations
sum(rate(chat_message_size_exceeded_total[5m])) by (type)
```

### AI Performance
```promql
# Token generation rate
rate(chat_ai_stream_tokens_total[1m]) * 60

# AI streaming latency (P95)
histogram_quantile(0.95, rate(chat_ai_stream_duration_seconds_bucket[5m]))

# Truncation rate
rate(chat_ai_response_truncations_total[5m])
```

### Database Performance
```promql
# Message save latency (P95)
histogram_quantile(0.95, rate(chat_message_save_duration_seconds_bucket[5m]))

# Message save rate
rate(chat_message_save_duration_seconds_count[5m])
```

---

## 🚨 Recommended Alerts

### Critical Alerts

```yaml
# High validation rejection rate (possible DoS)
- alert: HighValidationRejectionRate
  expr: rate(chat_message_validation_rejections_total[1m]) * 60 > 100
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "High validation rejection rate detected"
    description: "More than 100 messages rejected per minute for 2 minutes"

# No active connections (service down)
- alert: NoActiveConnections
  expr: chat_websocket_active_connections == 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "No active WebSocket connections"
    description: "No users connected for 5 minutes"

# High memory usage
- alert: HighMemoryUsage
  expr: go_memstats_alloc_bytes > 1e9  # 1GB
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High memory usage detected"
    description: "Memory usage above 1GB for 5 minutes"
```

### Performance Alerts

```yaml
# Slow message saves
- alert: SlowMessageSaves
  expr: histogram_quantile(0.95, rate(chat_message_save_duration_seconds_bucket[5m])) > 1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Database writes are slow"
    description: "P95 message save latency above 1 second"

# High AI streaming latency
- alert: HighAILatency
  expr: histogram_quantile(0.95, rate(chat_ai_stream_duration_seconds_bucket[5m])) > 60
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "AI streaming is slow"
    description: "P95 AI streaming latency above 60 seconds"
```

---

## 🔍 Troubleshooting

### Prometheus Not Scraping

1. Check Prometheus targets: http://localhost:9090/targets
2. Verify your app is running: `curl http://localhost:5555/metrics`
3. Check Docker network connectivity:
   ```bash
   docker exec -it hyperion-prometheus ping host.docker.internal
   ```

### Grafana Dashboard Empty

1. Verify Prometheus datasource is configured
2. Check that metrics are being scraped in Prometheus
3. Generate some traffic to your app to populate metrics
4. Adjust time range in Grafana (try "Last 5 minutes")

### Metrics Not Updating

1. Check scrape interval in `prometheus.yml` (default 15s)
2. Verify your app is exposing metrics: `curl http://localhost:5555/metrics`
3. Look for errors in Prometheus logs:
   ```bash
   docker logs hyperion-prometheus
   ```

---

## 📊 Production Deployment

### For Kubernetes

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
      - job_name: 'hyperion-chat'
        kubernetes_sd_configs:
          - role: pod
            namespaces:
              names:
                - hyperion
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_label_app]
            action: keep
            regex: hyperion-chat
          - source_labels: [__meta_kubernetes_pod_ip]
            target_label: __address__
            replacement: $1:5555
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
      volumes:
      - name: config
        configMap:
          name: prometheus-config
```

---

## 🎓 Next Steps

1. **Set up alerts** using Prometheus Alertmanager
2. **Configure retention** based on your storage needs
3. **Add more panels** to the Grafana dashboard
4. **Integrate with PagerDuty/Slack** for alert notifications
5. **Set up long-term storage** with Thanos or Cortex

---

## 📚 Resources

- **Prometheus Documentation**: https://prometheus.io/docs/
- **Grafana Documentation**: https://grafana.com/docs/
- **PromQL Tutorial**: https://prometheus.io/docs/prometheus/latest/querying/basics/
- **Grafana Dashboards**: https://grafana.com/grafana/dashboards/

---

## ✅ Verification Checklist

- [ ] Prometheus is scraping your app
- [ ] Grafana dashboard shows data
- [ ] All 14 panels are rendering
- [ ] Metrics update every 10-15 seconds
- [ ] Can query metrics in Prometheus UI
- [ ] Alerts are configured (optional)
- [ ] Dashboard is saved in Grafana

---

## 🎉 You're Done!

Your chat system now has **production-grade observability**! Monitor your metrics, set up alerts, and keep your system healthy.

For questions or issues, check the logs:
```bash
# Application logs
tail -f /tmp/hyper-metrics.log

# Prometheus logs
docker logs hyperion-prometheus

# Grafana logs
docker logs hyperion-grafana
```
