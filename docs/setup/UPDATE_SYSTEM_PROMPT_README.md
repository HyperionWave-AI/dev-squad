# System Prompt Update Guide

This guide explains how to update the Hyperion system prompt to enable MCP tool discovery capabilities.

## What's Being Added

The updated system prompt adds three new MCP tool discovery tools that Claude can use:

1. **discover_tools** - Discover external MCP tools using natural language queries
2. **get_tool_schema** - Get complete JSON schema for specific MCP tools
3. **execute_tool** - Execute external MCP tools by name with arguments

## Files Created

- `updated_system_prompt.md` - The new system prompt content with MCP tools
- `update_system_prompt.py` - Python script to update MongoDB (RECOMMENDED)
- `update_system_prompt.js` - MongoDB shell script alternative
- `update_prompt_via_api.sh` - Bash script with API instructions

## Quick Start (Recommended Method)

### Option 1: Python Script (Easiest)

```bash
# Install dependencies
pip3 install pymongo

# Run the update script
python3 update_system_prompt.py
```

This will:
- Connect to MongoDB at `localhost:27017` (or `$MONGO_URI` if set)
- Find all active system prompts
- Create new versions with MCP tool discovery capabilities
- Deactivate old versions

### Option 2: MongoDB Shell Script

```bash
# Run with mongosh
mongosh hyperion_db < update_system_prompt.js
```

### Option 3: Direct MongoDB Update

```bash
# Connect to MongoDB
mongosh hyperion_db

# Run this in the MongoDB shell
load('update_system_prompt.js')
```

## Environment Variables

- `MONGO_URI` - MongoDB connection string (default: `mongodb://localhost:27017`)

## What Changes

### Before
The system prompt restricted Claude to only coordinator tools:

```
❌ Tools I DON'T Have
❌ read_file
❌ write_file
❌ code_index_search
❌ discover_tools (not mentioned)
❌ get_tool_schema (not mentioned)
❌ execute_tool (not mentioned)
```

### After
The system prompt explicitly includes MCP discovery tools:

```
✅ Tools I Have Access To

### MCP Tool Discovery & Execution (NEW!)
- discover_tools - Discover external MCP tools using natural language
- get_tool_schema - Get complete JSON schema for MCP tools
- execute_tool - Execute external MCP tools by name
```

## Verification

After running the update:

1. **Restart your chat session** (refresh the UI)
2. Ask Claude: "What tools do you have access to?"
3. Claude should now list `discover_tools`, `get_tool_schema`, and `execute_tool`

## Example Usage

Once updated, Claude can use these tools:

### Discover External Tools
```
User: "Find tools for video processing"
Claude:
  → discover_tools({ query: "video processing" })
  → Returns: List of video-related MCP tools
```

### Get Tool Schema
```
User: "How do I use the video_transcribe tool?"
Claude:
  → get_tool_schema({ toolName: "video_transcribe" })
  → Returns: Full schema with parameters
```

### Execute External Tool
```
User: "Transcribe this video: https://example.com/video.mp4"
Claude:
  → execute_tool({
      toolName: "video_transcribe",
      args: { url: "https://example.com/video.mp4" }
    })
  → Returns: Transcription result
```

## Troubleshooting

### MongoDB Connection Failed
```bash
# Check if MongoDB is running
mongosh --eval "db.version()"

# If not running, start it:
brew services start mongodb-community
# or
docker run -d -p 27017:27017 mongo
```

### Python Dependencies Missing
```bash
pip3 install pymongo
```

### No Active System Prompts
If no active prompts exist, the script creates a default one for `test-user-id`. You may need to:
1. Create a user account in the UI
2. Log in and create a custom system prompt
3. Run this update script again

## Manual Verification

Check MongoDB directly:

```bash
mongosh hyperion_db --eval "db.system_prompt_versions.find({isActive: true}).pretty()"
```

You should see the new prompt with MCP tool discovery sections.

## Rollback

If you need to revert:

```bash
mongosh hyperion_db --eval "
  db.system_prompt_versions.updateMany(
    { version: { \$gt: 1 } },
    { \$set: { isActive: false } }
  );
  db.system_prompt_versions.updateMany(
    { version: 1 },
    { \$set: { isActive: true } }
  );
"
```

## Support

If you encounter issues:
1. Check MongoDB is running and accessible
2. Verify the `updated_system_prompt.md` file exists
3. Check logs for errors
4. Ensure the Hyperion server is stopped during update (optional but safer)
