#!/bin/bash

# Show AI request/response sequence from logs

LOGS_DIR="${1:-./logs}"

if [ ! -d "$LOGS_DIR" ]; then
    echo "❌ Directory $LOGS_DIR does not exist"
    echo "💡 Logs will be created automatically when the coordinator makes AI requests."
    exit 1
fi

# Count log files
REQ_COUNT=$(ls "$LOGS_DIR"/*.req.json 2>/dev/null | wc -l)

if [ "$REQ_COUNT" -eq 0 ]; then
    echo "⚠️  No log files found in $LOGS_DIR"
    echo "💡 Logs will be created automatically when the coordinator makes AI requests."
    exit 0
fi

echo "🔍 AI Request/Response Sequence Analysis"
echo "📁 Directory: $LOGS_DIR"
echo "📊 Total pairs: $REQ_COUNT"
echo ""
echo "========================================================================"

# Iterate through pairs in order
for req_file in "$LOGS_DIR"/*.req.json; do
    # Extract number from filename
    num=$(basename "$req_file" .req.json)
    res_file="$LOGS_DIR/$num.res.json"

    echo ""
    echo "🔸 Pair #$num"
    echo "------------------------------------------------------------------------"

    # Show request summary
    echo "📤 REQUEST:"
    echo "  File: $(basename "$req_file")"

    # Extract key info using jq if available, otherwise use grep
    if command -v jq &> /dev/null; then
        provider=$(jq -r '.provider' "$req_file" 2>/dev/null)
        model=$(jq -r '.options.model' "$req_file" 2>/dev/null)
        msg_count=$(jq '.msgContents | length' "$req_file" 2>/dev/null)
        tool_count=$(jq '.tools | length' "$req_file" 2>/dev/null)
        timestamp=$(jq -r '.timestamp' "$req_file" 2>/dev/null)

        echo "  Timestamp: $timestamp"
        echo "  Provider: $provider"
        echo "  Model: $model"
        echo "  Messages: $msg_count"
        echo "  Tools: $tool_count"

        # Show last message role
        last_role=$(jq -r '.msgContents[-1].role' "$req_file" 2>/dev/null)
        echo "  Last message role: $last_role"
    else
        # Fallback without jq
        echo "  (Install jq for detailed analysis)"
        grep -o '"provider":"[^"]*"' "$req_file" | head -1
        grep -o '"model":"[^"]*"' "$req_file" | head -1
    fi

    # Show response summary
    echo ""
    echo "📥 RESPONSE:"

    if [ ! -f "$res_file" ]; then
        echo "  ❌ Response file missing: $(basename "$res_file")"
        continue
    fi

    echo "  File: $(basename "$res_file")"

    if command -v jq &> /dev/null; then
        timestamp=$(jq -r '.timestamp' "$res_file" 2>/dev/null)
        error=$(jq -r '.error // empty' "$res_file" 2>/dev/null)
        content=$(jq -r '.response.choices[0].content // empty' "$res_file" 2>/dev/null)
        tool_calls=$(jq '.response.choices[0].tool_calls | length // 0' "$res_file" 2>/dev/null)
        stop_reason=$(jq -r '.response.choices[0].stop_reason // empty' "$res_file" 2>/dev/null)

        echo "  Timestamp: $timestamp"

        if [ -n "$error" ]; then
            echo "  ❌ Error: $error"
        else
            echo "  Stop reason: $stop_reason"
            echo "  Tool calls: $tool_calls"

            # Show content preview
            if [ -n "$content" ]; then
                content_preview=$(echo "$content" | head -c 100)
                echo "  Content preview: ${content_preview}..."
            fi

            # Show tool call names if any
            if [ "$tool_calls" -gt 0 ]; then
                echo "  Tool call names:"
                jq -r '.response.choices[0].tool_calls[].function_call.name' "$res_file" 2>/dev/null | while read -r tool; do
                    echo "    - $tool"
                done
            fi
        fi
    else
        # Fallback without jq
        if grep -q '"error"' "$res_file"; then
            echo "  ❌ Contains error"
        else
            echo "  ✅ Success"
        fi
    fi

    echo ""
done

echo "========================================================================"
echo ""
echo "✅ Analysis complete"
echo ""
echo "💡 Tips:"
echo "  - Use 'jq' to view full JSON: jq . logs/1.req.json"
echo "  - Compare request/response: diff <(jq . logs/1.req.json) <(jq . logs/1.res.json)"
echo "  - View specific field: jq '.msgContents' logs/1.req.json"
echo ""
