# Quick Start: Enable MCP Tool Discovery

## TL;DR

```bash
# 1. Install Python dependencies
pip3 install pymongo

# 2. Run the update script
cd /Users/maxmednikov/MaxSpace/hyper
python3 update_system_prompt.py

# 3. Restart your chat session (refresh browser)

# 4. Test it
# Ask Claude: "What tools do you have access to?"
# Should now include: discover_tools, get_tool_schema, execute_tool
```

## What This Does

Enables Claude in Hyperion chat to:
- 🔍 **discover_tools** - Find external MCP tools
- 📋 **get_tool_schema** - Get tool documentation
- ▶️ **execute_tool** - Run external tools

## Why This Works

The tools were already registered in the backend, but the system prompt told Claude to ignore them. This update changes the system prompt to explicitly allow these tools.

## Verification

After update, ask Claude:
```
"Can you use discover_tools?"
```

Expected: ✅ "Yes, I can discover external MCP tools..."

## Files

- ✅ `updated_system_prompt.md` - New prompt
- ✅ `update_system_prompt.py` - Update script
- ✅ `UPDATE_SYSTEM_PROMPT_README.md` - Full docs
- ✅ `MCP_TOOLS_ENABLED_SUMMARY.md` - Technical details

## Troubleshooting

**MongoDB not running?**
```bash
brew services start mongodb-community
```

**Python missing pymongo?**
```bash
pip3 install pymongo
```

**Still not working?**
- Clear browser cache
- Create new chat session
- Check MongoDB: `mongosh hyperion_db --eval "db.system_prompt_versions.find({isActive:true}).count()"`

---

**Ready?** Run: `python3 update_system_prompt.py` 🚀
