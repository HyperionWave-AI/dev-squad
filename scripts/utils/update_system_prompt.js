// MongoDB script to update the active system prompt with MCP tool discovery tools
// Run with: mongosh hyperion_db update_system_prompt.js

// Read the new system prompt content
const fs = require('fs');
const newPrompt = fs.readFileSync('/Users/maxmednikov/MaxSpace/hyper/updated_system_prompt.md', 'utf8');

// Get the database
const db = db.getSiblingDB('hyperion_db');

// Find all active system prompt versions
const activeVersions = db.system_prompt_versions.find({ isActive: true }).toArray();

print(`Found ${activeVersions.length} active system prompt version(s)`);

// Update each active version
activeVersions.forEach(version => {
  print(`\nUpdating system prompt for user: ${version.userId}, company: ${version.companyId}`);
  print(`Current version: ${version.version}`);

  // Create a new version with the updated prompt
  const newVersion = {
    userId: version.userId,
    companyId: version.companyId,
    version: version.version + 1,
    prompt: newPrompt,
    description: "Added MCP tool discovery capabilities (discover_tools, get_tool_schema, execute_tool)",
    isActive: true,
    createdAt: new Date(),
    updatedAt: new Date()
  };

  // Deactivate the old version
  db.system_prompt_versions.updateOne(
    { _id: version._id },
    { $set: { isActive: false, updatedAt: new Date() } }
  );

  // Insert the new version
  const result = db.system_prompt_versions.insertOne(newVersion);

  print(`✓ Created new version ${newVersion.version} (ID: ${result.insertedId})`);
  print(`✓ Deactivated old version ${version.version}`);
});

// If no active versions exist, create a default one for the test user
if (activeVersions.length === 0) {
  print("\nNo active system prompts found. Creating default for test user...");

  const defaultVersion = {
    userId: "test-user-id",
    companyId: "test-company-id",
    version: 1,
    prompt: newPrompt,
    description: "Initial system prompt with MCP tool discovery capabilities",
    isActive: true,
    createdAt: new Date(),
    updatedAt: new Date()
  };

  const result = db.system_prompt_versions.insertOne(defaultVersion);
  print(`✓ Created default system prompt (ID: ${result.insertedId})`);
}

print("\n✅ System prompt update complete!");
print("\nThe updated prompt now includes:");
print("  - discover_tools - Discover external MCP tools");
print("  - get_tool_schema - Get tool schemas");
print("  - execute_tool - Execute external MCP tools");
