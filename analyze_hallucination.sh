#!/bin/bash

# Analyze AI hallucination in logs

echo "🔍 AI Hallucination Analysis"
echo "======================================"
echo ""

# Find the last 10 request/response pairs
cd /Users/maxmednikov/MaxSpace/hyper/logs

LAST_REQ=$(ls -1 *.req.json | sort -V | tail -1 | sed 's/.req.json//')

echo "📊 Analyzing last 10 pairs ending at #$LAST_REQ"
echo ""

START=$((LAST_REQ - 9))
if [ $START -lt 1 ]; then START=1; fi

for i in $(seq $START $LAST_REQ); do
    if [ ! -f "$i.req.json" ] || [ ! -f "$i.res.json" ]; then
        continue
    fi

    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔸 Pair #$i"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # Request analysis
    MSG_COUNT=$(jq '.msgContents | length' $i.req.json)
    TOOL_COUNT=$(jq '.tools | length' $i.req.json)

    echo "📤 REQUEST:"
    echo "   Messages: $MSG_COUNT"
    echo "   Tools available: $TOOL_COUNT"

    # Last 3 messages
    echo "   Last 3 message roles:"
    jq -r '.msgContents[-3:] | .[] | "     - \(.role): \([.parts[] | .type] | join(","))"' $i.req.json

    # Check for tool_call_response without matching tool_call
    echo ""
    echo "   🔍 Tool Call/Response Matching:"

    # Get all tool calls (AI messages with tool_call parts)
    TOOL_CALLS=$(jq -r '.msgContents[] | select(.role == "ai") | .parts[] | select(.type == "tool_call") | .id' $i.req.json | sort)

    # Get all tool responses
    TOOL_RESPONSES=$(jq -r '.msgContents[] | select(.role == "human") | .parts[] | select(.type == "tool_call_response") | .tool_call_id' $i.req.json | sort)

    # Count them
    CALL_COUNT=$(echo "$TOOL_CALLS" | grep -c "functions" || echo "0")
    RESP_COUNT=$(echo "$TOOL_RESPONSES" | grep -c "functions" || echo "0")

    echo "     Tool calls in history: $CALL_COUNT"
    echo "     Tool responses in history: $RESP_COUNT"

    if [ "$CALL_COUNT" != "$RESP_COUNT" ]; then
        echo "     ⚠️  MISMATCH! Calls != Responses"
    else
        echo "     ✅ Balanced"
    fi

    # Response analysis
    echo ""
    echo "📥 RESPONSE:"

    RESP_TOOL_CALLS=$(jq '.response.choices[0].tool_calls | length' $i.res.json)
    STOP_REASON=$(jq -r '.response.choices[0].stop_reason // "none"' $i.res.json)
    CONTENT=$(jq -r '.response.choices[0].content // ""' $i.res.json | head -c 80)

    echo "   Stop reason: $STOP_REASON"
    echo "   Tool calls requested: $RESP_TOOL_CALLS"

    if [ "$RESP_TOOL_CALLS" -gt 0 ]; then
        echo "   Requested tools:"
        jq -r '.response.choices[0].tool_calls[] | "     - \(.function_call.name) (ID: \(.id))"' $i.res.json
    fi

    if [ -n "$CONTENT" ]; then
        echo "   Content preview: $CONTENT..."
    fi

    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "✅ Analysis complete"
