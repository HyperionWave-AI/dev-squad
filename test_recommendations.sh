#!/bin/bash

echo "=========================================="
echo "  PROACTIVE RECOMMENDATIONS TEST"
echo "=========================================="
echo

echo "1. Searching for 'MongoDB timeout' lessons:"
curl -s "http://localhost:4097/api/v1/reflection/search?q=MongoDB%20timeout&limit=3" | jq '.count, .lessons[].data.patternName'
echo

echo "2. Searching for 'TypeScript' lessons:"
curl -s "http://localhost:4097/api/v1/reflection/search?q=TypeScript&limit=3" | jq '.count, .lessons[].data.patternName'
echo

echo "3. Searching for 'hardcoding' lessons:"
curl -s "http://localhost:4097/api/v1/reflection/search?q=hardcoding&limit=3" | jq '.count, .lessons[].data.patternName'
echo

echo "4. Total lessons in system:"
curl -s "http://localhost:4097/api/v1/reflection/lessons" | jq '.count'
echo

echo "=========================================="
echo "All MCP reflection tools available:"
echo "  - reflection_record_decision"
echo "  - reflection_record_outcome"
echo "  - reflection_extract_lesson"
echo "  - reflection_suggest_lesson_from_error (NEW)"
echo "  - reflection_query_relevant_lessons (NEW)"
echo "=========================================="
