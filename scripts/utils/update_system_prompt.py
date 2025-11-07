#!/usr/bin/env python3
"""
Script to update the Hyperion system prompt with MCP tool discovery capabilities.
This script directly updates MongoDB to add discover_tools, get_tool_schema, and execute_tool.
"""

import os
import sys
from datetime import datetime
from pymongo import MongoClient
from bson.objectid import ObjectId

# MongoDB connection settings
MONGO_URI = os.getenv('MONGO_URI', 'mongodb://localhost:27017')
DATABASE_NAME = 'hyperion_db'

def read_new_prompt():
    """Read the updated system prompt from file."""
    prompt_file = os.path.join(os.path.dirname(__file__), 'updated_system_prompt.md')
    if not os.path.exists(prompt_file):
        print(f"ERROR: Prompt file not found: {prompt_file}")
        sys.exit(1)

    with open(prompt_file, 'r') as f:
        return f.read()

def update_system_prompts():
    """Update all active system prompts in MongoDB."""
    try:
        # Connect to MongoDB
        print(f"Connecting to MongoDB at {MONGO_URI}...")
        client = MongoClient(MONGO_URI)
        db = client[DATABASE_NAME]
        collection = db['system_prompt_versions']

        # Read the new prompt
        new_prompt = read_new_prompt()
        print(f"Read new prompt ({len(new_prompt)} characters)")

        # Find all active system prompt versions
        active_versions = list(collection.find({'isActive': True}))
        print(f"\nFound {len(active_versions)} active system prompt version(s)")

        if len(active_versions) == 0:
            print("\nNo active system prompts found. Creating default for test user...")

            # Create default version for test user
            default_version = {
                'userId': 'test-user-id',
                'companyId': 'test-company-id',
                'version': 1,
                'prompt': new_prompt,
                'description': 'Initial system prompt with MCP tool discovery capabilities',
                'isActive': True,
                'createdAt': datetime.utcnow(),
                'updatedAt': datetime.utcnow()
            }

            result = collection.insert_one(default_version)
            print(f"✓ Created default system prompt (ID: {result.inserted_id})")
        else:
            # Update each active version
            for version in active_versions:
                user_id = version.get('userId', 'unknown')
                company_id = version.get('companyId', 'unknown')
                current_version = version.get('version', 0)

                print(f"\nUpdating system prompt for user: {user_id}, company: {company_id}")
                print(f"Current version: {current_version}")

                # Create new version
                new_version_num = current_version + 1
                new_version = {
                    'userId': user_id,
                    'companyId': company_id,
                    'version': new_version_num,
                    'prompt': new_prompt,
                    'description': 'Added MCP tool discovery capabilities (discover_tools, get_tool_schema, execute_tool)',
                    'isActive': True,
                    'createdAt': datetime.utcnow(),
                    'updatedAt': datetime.utcnow()
                }

                # Deactivate old version
                collection.update_one(
                    {'_id': version['_id']},
                    {'$set': {'isActive': False, 'updatedAt': datetime.utcnow()}}
                )

                # Insert new version
                result = collection.insert_one(new_version)

                print(f"✓ Created new version {new_version_num} (ID: {result.inserted_id})")
                print(f"✓ Deactivated old version {current_version}")

        print("\n" + "="*60)
        print("✅ System prompt update complete!")
        print("="*60)
        print("\nThe updated prompt now includes:")
        print("  ✓ discover_tools - Discover external MCP tools")
        print("  ✓ get_tool_schema - Get tool schemas")
        print("  ✓ execute_tool - Execute external MCP tools")
        print("\nRestart your chat session for changes to take effect.")

        client.close()

    except Exception as e:
        print(f"\n❌ ERROR: {e}")
        sys.exit(1)

if __name__ == '__main__':
    print("="*60)
    print("Hyperion System Prompt Update Script")
    print("="*60)
    print()

    update_system_prompts()
