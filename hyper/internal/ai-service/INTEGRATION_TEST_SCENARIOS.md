# ToolResultProcessor Integration Test Scenarios

## Overview
This document describes real-world chat scenarios used to verify token reduction and context tracking functionality.

## Test Scenarios

### Scenario 1: Simple Tool Execution
**Description**: User asks for a simple tool execution (e.g., read a file)

**Expected Metrics**:
- Input Tokens: ~50-100
- Output Tokens: ~30-50
- Tool Result Tokens: ~200-500 (file content)
- Processed Tokens: ~60-150 (after truncation/summarization)
- Token Reduction: ~60-70%

**Verification**:
- ✅ Tool result is processed before sending to LLM
- ✅ Token reduction is logged
- ✅ Context size is tracked

### Scenario 2: Multiple Tool Calls
**Description**: User asks for multiple sequential tool calls

**Expected Metrics**:
- Input Tokens: ~100-150
- Output Tokens: ~50-100
- Tool Result Tokens: ~500-1000 (multiple results)
- Processed Tokens: ~150-300 (after processing)
- Token Reduction: ~65-75%

**Verification**:
- ✅ Each tool result is processed independently
- ✅ Cumulative token reduction is calculated
- ✅ Context size grows appropriately

### Scenario 3: Large Tool Results
**Description**: User triggers a tool that returns large output (e.g., code search)

**Expected Metrics**:
- Input Tokens: ~50-100
- Output Tokens: ~30-50
- Tool Result Tokens: ~1000-5000 (large output)
- Processed Tokens: ~100-300 (after suppression/truncation)
- Token Reduction: ~80-95%

**Verification**:
- ✅ Large results are suppressed or truncated
- ✅ Helpful message is shown to user
- ✅ Significant token reduction is achieved

### Scenario 4: Context Window Management
**Description**: Long conversation with multiple exchanges

**Expected Metrics**:
- Input Tokens: ~200-300
- Output Tokens: ~100-150
- Tool Result Tokens: ~500-1000
- Processed Tokens: ~150-300
- Context Size: ~10000-50000 bytes
- Sliding Window Applied: Yes (when > 50% of max)

**Verification**:
- ✅ Context size is monitored
- ✅ Sliding window is applied when needed
- ✅ Token reduction is maintained across exchanges

## Running Test Scenarios

### Manual Testing
1. Start the server: `make run`
2. Connect to WebSocket: `ws://localhost:8080/api/v1/chat/stream?sessionId=<id>`
3. Send test messages and observe metrics in logs

### Automated Testing
Run the integration tests:
```bash
go test -v ./hyper/internal/ai-service -run TestContextTracker
```

## Metrics Collection

### Log Output Format
```
Token Usage Metrics - Session: <id> | Input: 100 | Output: 50 | Tool Results: 300 → 90 (70% reduction) | Context: 8000 bytes | Duration: 150ms
```

### Metrics Map
```json
{
  "input_tokens": 100,
  "output_tokens": 50,
  "tool_result_tokens": 300,
  "processed_tokens": 90,
  "token_reduction": 70.0,
  "context_size": 8000,
  "duration_ms": 150
}
```

## Performance Targets

| Metric | Target | Actual |
|--------|--------|--------|
| Token Reduction | 60-80% | TBD |
| Context Size Growth | <50% per exchange | TBD |
| Processing Overhead | <10ms | TBD |
| Memory Usage | <100MB | TBD |

## Known Issues & Limitations

1. **Token Counting**: Approximate token counts based on character length
   - Actual token counts may vary by model
   - Consider using model-specific tokenizers for accuracy

2. **Context Size Calculation**: Simplified calculation
   - Doesn't account for encoding overhead
   - May underestimate actual context size

3. **Sliding Window**: Basic implementation
   - May not be optimal for all conversation patterns
   - Consider implementing more sophisticated algorithms

## Future Improvements

1. **Accurate Token Counting**
   - Integrate with model-specific tokenizers
   - Track actual token usage from API responses

2. **Advanced Context Management**
   - Implement priority-based message retention
   - Add semantic similarity-based summarization

3. **Performance Optimization**
   - Cache processed results
   - Implement parallel processing for multiple tools

4. **Monitoring & Alerting**
   - Add metrics to Prometheus
   - Set up alerts for excessive token usage
   - Create dashboards for visualization
