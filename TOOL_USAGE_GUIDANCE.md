# Tool Usage Guidance for Chat System

## Overview
This guide provides comprehensive instructions for using available tools effectively in the chat system. Tools are organized by purpose with clear use cases, best practices, and common patterns.

---

## 📋 TASK & WORKFLOW MANAGEMENT

### Human Task Creation & Tracking
**Tools:** `coordinator_create_human_task`, `coordinator_list_human_tasks`

**Purpose:** Create and track user requests as formal tasks in the system.

**When to use:**
- User makes a request that needs tracking
- Work requires visibility and status monitoring
- Multiple steps need coordination

**Best practices:**
```
1. Always create a human task for user requests
2. Use descriptive prompts that capture the full context
3. Check for similar tasks before creating duplicates
4. Reference the human task ID in agent tasks
```

**Example:**
```
coordinator_create_human_task(
  prompt: "Implement user authentication with JWT tokens"
)
```

---

### Agent Task Creation & Management
**Tools:** `create_agent_task`, `list_agent_tasks`, `coordinator_get_agent_task`

**Purpose:** Assign work to AI agents with detailed context and tracking.

**When to use:**
- Breaking down human tasks into agent work
- Assigning specific development tasks
- Tracking agent progress with TODOs

**Key parameters:**
- `agentName`: Name of the agent (e.g., "Developer-Backend")
- `role`: Agent's responsibility (e.g., "Backend Development")
- `todos`: Array of specific tasks to complete
- `contextSummary`: Detailed context from code search
- `filesModified`: File paths discovered from code search
- `complexityAnalysis`: Required when complexity analysis is enabled

**Best practices:**
```
1. Use code_index_search FIRST to discover files
2. Include file paths and line numbers in contextSummary
3. Break work into clear, actionable TODOs
4. Provide complexity analysis when available
5. Reference parent human task ID
```

**Example:**
```
create_agent_task(
  agentName: "Developer-Backend",
  role: "Go Backend Development",
  todos: [
    "Add JWT validation middleware to auth.go",
    "Create token refresh endpoint",
    "Add unit tests for auth flow"
  ],
  contextSummary: "Add JWT auth to api/auth.go lines 42-85...",
  filesModified: ["./api/auth.go", "./middleware/jwt.go"]
)
```

---

### Task Status & Progress Tracking
**Tools:** `coordinator_update_task_status`, `coordinator_update_todo_status`, `mcp_hyperion_tasks_api_task_complete`

**Purpose:** Update task progress and mark work as complete.

**When to use:**
- Starting work on a task
- Completing individual TODOs
- Marking entire tasks as done
- Blocking on dependencies

**Status values:**
- `pending`: Not started
- `in_progress`: Currently being worked on
- `completed`: Finished successfully
- `blocked`: Waiting on something

**Best practices:**
```
1. Update status as work progresses
2. Add notes explaining status changes
3. Mark TODOs complete as they finish
4. Use "blocked" status with explanation
```

---

### Task Guidance & Notes
**Tools:** `coordinator_add_task_prompt_notes`, `coordinator_update_task_prompt_notes`, `coordinator_clear_task_prompt_notes`

**Purpose:** Add human guidance to agent tasks.

**When to use:**
- Providing additional context to agents
- Clarifying requirements
- Sharing important patterns or examples
- Updating guidance based on progress

**Best practices:**
```
1. Use markdown formatting for clarity
2. Include code examples when relevant
3. Reference specific files and line numbers
4. Update notes if requirements change
```

---

## 🔍 CODE DISCOVERY & ANALYSIS

### Code Search
**Tools:** `code_index_search`, `code_index_get_full_content`, `code_index_status`

**Purpose:** Find relevant code and retrieve full content efficiently.

**When to use:**
- Discovering files to modify
- Finding patterns and examples
- Understanding existing implementations
- Locating specific functions or classes

**Search parameters:**
- `query`: Natural language description (e.g., "authentication logic")
- `limit`: Number of results (default: 10, max: 50)
- `responseMode`: Control token usage (summary/preview/full)
- `retrieve`: Content retrieval mode (chunk/chunk-s/chunk-m/chunk-l/chunk-xl/full)
- `functionName`: Filter by function pattern
- `className`: Filter by exact class name
- `nodeType`: Filter by code element type (function/class/method/interface/import)

**Best practices:**
```
1. Start with natural language queries
2. Use responseMode='summary' for initial exploration
3. Use retrieve='chunk' for large files
4. Filter by nodeType for specific searches
5. Use code_index_get_full_content for complete file content
6. Check code_index_status to verify indexing
```

**Workflow:**
```
1. code_index_search(query: "find auth middleware")
   → Returns summaries of matching files
2. Review results and pick 2-3 most relevant
3. code_index_get_full_content(filePath: "./api/auth.go")
   → Get complete file for detailed analysis
4. Use findings in agent task context
```

---

### Complexity Analysis
**Tools:** `analyze_complexity`

**Purpose:** Assess task complexity and determine if work should be split.

**When to use:**
- Before creating agent tasks
- For significant code changes
- When complexity analysis mode is enabled (MANDATORY)

**Complexity factors:**
- File count
- File size
- Cross-squad impact
- Architectural scope
- Estimated line changes

**Output:**
- `overallScore`: 0.0-1.0 complexity score
- `level`: low/medium/high/extreme
- `shouldSplit`: Whether to break into smaller tasks
- `reasoning`: Explanation of complexity

**Best practices:**
```
1. Always call before creating agent tasks (when enabled)
2. Pass result to create_agent_task in complexityAnalysis field
3. If shouldSplit=true, break into multiple agent tasks
4. Use complexity level to set priority and timeline
```

---

## 📁 FILE & STORAGE MANAGEMENT

### File Operations
**Tools:** `read_file`, `write_file`, `list_directory`, `apply_patch`

**Purpose:** Read, write, and manage files in the project.

**When to use:**
- Reading source files for analysis
- Creating new files
- Modifying existing files
- Listing directory contents

**Best practices:**
```
1. Always read file first before editing
2. Use apply_patch for precise changes
3. Use write_file for new files
4. Preserve formatting and style
5. Use relative paths from project root
```

**Example:**
```
# Read file
read_file("./api/auth.go")

# Write new file
write_file(
  path: "./api/jwt.go",
  content: "package api\n\nfunc ValidateToken() {...}"
)

# Apply patch
apply_patch(
  path: "./api/auth.go",
  patch: "--- a/api/auth.go\n+++ b/api/auth.go\n..."
)
```

---

### Storage API (Hyperion)
**Tools:** `mcp_hyperion_storage_api_*` (list_directory, create_directory, delete_file, etc.)

**Purpose:** Manage files in Hyperion storage system.

**When to use:**
- Uploading generated files
- Managing media assets
- Organizing project files
- Sharing files

**Common operations:**
```
mcp_hyperion_storage_api_list_directory(path: "/uploads")
mcp_hyperion_storage_api_create_directory(path: "/uploads/2024")
mcp_hyperion_storage_api_delete_file(path: "hyperion://files/old.txt")
mcp_hyperion_storage_api_copy_file(source: "hyperion://files/a.txt", destination: "/backup")
mcp_hyperion_storage_api_move_file(source: "hyperion://files/a.txt", destination: "/archive")
mcp_hyperion_storage_api_share_public_link(path: "hyperion://files/report.pdf")
```

---

## 🎬 VIDEO & MEDIA GENERATION

### Video Generation
**Tools:** `mcp_hyperion_google_mcp_google_generate_video`, `mcp_hyperion_hedra_mcp_hedra_create_video`, `mcp_hyperion_heygen_mcp_heygen_create_video`

**Purpose:** Generate videos using different AI providers.

**Provider comparison:**
| Provider | Best For | Duration | Quality |
|----------|----------|----------|---------|
| Google Veo 3 | Text-to-video, natural scenes | Max 15s | 720p/1080p |
| Hedra | Lip-sync, talking videos | Flexible | 540p/720p |
| HeyGen | Avatar videos, templates | Flexible | Professional |

**When to use:**
- Google Veo: General video generation from prompts
- Hedra: Lip-sync videos with audio
- HeyGen: Avatar-based videos

**Best practices:**
```
1. Use ASYNC workflow - returns taskId immediately
2. Poll task status with appropriate task_get tool
3. Store results in Hyperion storage
4. Specify duration and resolution upfront
5. Use reference images for consistency
```

**Example:**
```
# Generate video with Google Veo
mcp_hyperion_google_mcp_google_generate_video(
  prompt: "A person walking through a forest",
  duration: 8,
  resolution: "720p"
)
→ Returns taskId

# Check status
mcp_hyperion_google_mcp_google_task_get(taskId: "...", waitSeconds: 30)
→ Returns completed video URI: hyperion://files/...
```

---

### Image Generation
**Tools:** `mcp_hyperion_hedra_mcp_hedra_create_image`, `mcp_hyperion_google_mcp_google_generate_image`, `mcp_hyperion_black_forest_labs_mcp_bfl_text_to_image`

**Purpose:** Generate images using different AI models.

**Provider comparison:**
| Provider | Model | Best For | Quality |
|----------|-------|----------|---------|
| Hedra | Multiple | General images | Good |
| Google | Gemini 2.5 | Detailed, complex | Excellent |
| Black Forest Labs | FLUX | Artistic, high-quality | Premium |

**When to use:**
- Hedra: Quick image generation
- Google: Detailed, complex images
- BFL: High-quality, artistic images

**Best practices:**
```
1. Use ASYNC workflow for most providers
2. Specify aspect ratio and resolution
3. Use detailed, specific prompts
4. Store results in Hyperion storage
5. Poll status regularly
```

---

### Audio & Speech Generation
**Tools:** `mcp_hyperion_google_mcp_google_generate_speech`, `mcp_hyperion_openai_mcp_transcribe_audio`

**Purpose:** Generate speech from text or transcribe audio.

**When to use:**
- Creating voiceovers
- Generating narration
- Transcribing audio files
- Multi-language speech

**Best practices:**
```
1. Use google_generate_speech for TTS
2. Specify voice ID and language
3. Use stylePrompt for natural delivery
4. Use transcribe_audio for audio-to-text
5. Store audio in Hyperion storage
```

**Example:**
```
# Generate speech
mcp_hyperion_google_mcp_google_generate_speech(
  text: "Welcome to our platform",
  voiceId: "en-US-Neural2-A",
  stylePrompt: "speak clearly and professionally"
)
→ Returns hyperion://files/speech.wav

# Transcribe audio
mcp_hyperion_openai_mcp_transcribe_audio(
  audioFile: "hyperion://files/recording.mp3"
)
→ Returns transcription text
```

---

### Video Processing & Editing
**Tools:** `mcp_hyperion_ffmpeg_mcp_*` (trim, resize, merge, extract_audio, add_subtitles, etc.)

**Purpose:** Process and edit videos with FFmpeg.

**Common operations:**
```
# Trim video
mcp_hyperion_ffmpeg_mcp_ffmpeg_trim_video(
  inputResourcePath: "hyperion://files/video.mp4",
  startTime: "00:00:05",
  endTime: "00:00:30"
)

# Resize video
mcp_hyperion_ffmpeg_mcp_ffmpeg_resize_video(
  inputResourcePath: "hyperion://files/video.mp4",
  format: "vertical",
  blurBackgroundMode: "enabled"
)

# Extract audio
mcp_hyperion_ffmpeg_mcp_ffmpeg_extract_audio(
  inputResourcePath: "hyperion://files/video.mp4",
  format: "mp3"
)

# Merge audio and video
mcp_hyperion_ffmpeg_mcp_ffmpeg_merge_audio_video(
  videoResourcePath: "hyperion://files/video.mp4",
  audioResourcePath: "hyperion://files/audio.mp3"
)

# Add subtitles
mcp_hyperion_ffmpeg_mcp_ffmpeg_add_subtitles(
  videoResourcePath: "hyperion://files/video.mp4",
  subtitleResourcePath: "hyperion://files/subs.srt",
  burnIn: true
)

# Concatenate videos
mcp_hyperion_ffmpeg_mcp_ffmpeg_video_concat(
  video1URI: "hyperion://files/intro.mp4",
  video2URI: "hyperion://files/main.mp4"
)

# Add overlays (text, images, HTML)
mcp_hyperion_ffmpeg_mcp_ffmpeg_add_overlay(
  inputUri: "hyperion://files/video.mp4",
  overlays: [
    {
      type: "text",
      text: "My Video",
      x: 100,
      y: 50,
      fontSize: 48
    }
  ]
)

# Optimize for web
mcp_hyperion_ffmpeg_mcp_ffmpeg_optimize_for_web(
  inputResourcePath: "hyperion://files/video.mp4",
  preset: "medium"
)
```

**Best practices:**
```
1. All FFmpeg operations are ASYNC
2. Poll task status regularly
3. Use appropriate formats for output
4. Chain operations efficiently
5. Store results in Hyperion storage
```

---

### Image Processing
**Tools:** `mcp_hyperion_ffmpeg_mcp_ffmpeg_resize_image`, `mcp_hyperion_ffmpeg_mcp_ffmpeg_add_text_overlay`

**Purpose:** Resize and edit images.

**Common operations:**
```
# Resize image with blur background
mcp_hyperion_ffmpeg_mcp_ffmpeg_resize_image(
  inputResourcePath: "hyperion://files/photo.jpg",
  format: "landscape",
  blurBackgroundMode: "enabled",
  cropPercentage: 0.8
)

# Add text overlay
mcp_hyperion_ffmpeg_mcp_ffmpeg_add_text_overlay(
  storageUri: "hyperion://files/image.jpg",
  textElements: [
    {
      text: "Special Offer!",
      x: 100,
      y: 50,
      fontSize: 48,
      fontColor: "white"
    }
  ]
)
```

---

## 🤖 AI & CONTENT ANALYSIS

### Document Analysis
**Tools:** `mcp_hyperion_google_mcp_google_understand_document`, `mcp_hyperion_google_mcp_google_detect_document_text`

**Purpose:** Extract and analyze document content.

**When to use:**
- Extracting text from PDFs
- Analyzing document structure
- OCR for scanned documents
- Understanding document content

**Best practices:**
```
1. Use understand_document for general analysis
2. Use detect_document_text for OCR/text extraction
3. Provide optional custom prompts for specific analysis
4. Store results for reference
```

---

### Image Analysis
**Tools:** `mcp_hyperion_google_mcp_google_understand_image`, `mcp_hyperion_google_mcp_google_detect_text_regions`

**Purpose:** Analyze image content and detect text.

**When to use:**
- Understanding image content
- Detecting text in images
- Analyzing visual elements
- Extracting text regions with coordinates

**Best practices:**
```
1. Use understand_image for general analysis
2. Use detect_text_regions for text extraction
3. Provide custom prompts for specific analysis
4. Use minConfidence to filter results
```

---

### Video Analysis
**Tools:** `mcp_hyperion_google_mcp_google_understand_video`, `mcp_hyperion_ffmpeg_mcp_ffmpeg_detect_faces`, `mcp_hyperion_ffmpeg_mcp_ffmpeg_get_video_info`

**Purpose:** Analyze video content and extract metadata.

**When to use:**
- Understanding video content
- Detecting faces in video
- Extracting video metadata
- Analyzing frame content

**Best practices:**
```
1. Use understand_video for content analysis
2. Use detect_faces for face detection
3. Use get_video_info for technical metadata
4. Specify frameInterval for face detection
```

---

### Audio Analysis
**Tools:** `mcp_hyperion_google_mcp_google_understand_audio`

**Purpose:** Transcribe and analyze audio content.

**When to use:**
- Understanding audio content
- Extracting key information
- Analyzing speech patterns
- Generating summaries

**Best practices:**
```
1. Provide optional custom prompts
2. Use for both transcription and analysis
3. Store results for reference
```

---

### Language Detection & Translation
**Tools:** `mcp_hyperion_openai_mcp_detect_language`, `mcp_hyperion_openai_mcp_translate_text`

**Purpose:** Detect language and translate text.

**When to use:**
- Identifying text language
- Translating content
- Multi-language support
- Language-specific processing

**Best practices:**
```
1. Detect language before translation
2. Specify target language clearly
3. Use appropriate model for quality
4. Store translations for reuse
```

---

## 🔗 PROCESS & WORKFLOW AUTOMATION

### Process Management
**Tools:** `mcp_hyperion_documents_api_process_*` (create, get, list, update, trigger, etc.)

**Purpose:** Create and manage business processes.

**When to use:**
- Defining repeatable workflows
- Automating multi-step procedures
- Tracking process execution
- Managing process instances

**Key operations:**
```
# Create process
mcp_hyperion_documents_api_process_create(
  name: "Content Creation Workflow",
  description: "Multi-step content creation process",
  procedures: [
    {
      stepNumber: 1,
      name: "Research",
      description: "Research topic",
      assigned: {type: "human", role: "researcher"},
      taskTemplate: "Research {{topic}} and compile findings"
    },
    ...
  ]
)

# Trigger process
mcp_hyperion_documents_api_process_trigger(
  processId: "...",
  execution_parameters: {
    topic: "AI in Healthcare",
    targetAudience: "Healthcare Professionals"
  }
)

# Get process details
mcp_hyperion_documents_api_process_get(processId: "...")

# List processes
mcp_hyperion_documents_api_process_list()

# Update process
mcp_hyperion_documents_api_process_update(
  processId: "...",
  name: "Updated Name",
  description: "Updated description"
)
```

---

### Process Instance Management
**Tools:** `mcp_hyperion_documents_api_instance_*` (get, list, pause, resume, cancel, etc.)

**Purpose:** Manage running process instances.

**When to use:**
- Monitoring process execution
- Pausing/resuming processes
- Checking instance status
- Managing context data

**Key operations:**
```
# Get instance details
mcp_hyperion_documents_api_instance_get(instanceId: "...")

# Get instance status
mcp_hyperion_documents_api_instance_get_status(instanceId: "...")

# Pause instance
mcp_hyperion_documents_api_instance_pause(instanceId: "...")

# Resume instance
mcp_hyperion_documents_api_instance_resume(instanceId: "...")

# Cancel instance
mcp_hyperion_documents_api_instance_cancel(instanceId: "...")

# Get execution logs
mcp_hyperion_documents_api_instance_get_logs(instanceId: "...", limit: 50)

# Get step details
mcp_hyperion_documents_api_instance_get_step_details(
  instanceId: "...",
  stepNumber: 1
)
```

---

### Process Context Management
**Tools:** `mcp_hyperion_documents_api_instance_context_*`

**Purpose:** Manage context data for process instances.

**When to use:**
- Storing execution parameters
- Passing data between steps
- Tracking process state
- Managing dynamic variables

**Best practices:**
```
1. Set context at instance creation
2. Update context as process progresses
3. Use context for template variables
4. Clear context when no longer needed
```

---

## 💬 CHAT & COMMUNICATION

### Conversation Management
**Tools:** `mcp_hyperion_chat_api_conversation_*` (create, get, list, find, delete)

**Purpose:** Manage chat conversations.

**When to use:**
- Creating new conversations
- Retrieving conversation history
- Finding specific conversations
- Managing conversation metadata

**Best practices:**
```
1. Create conversations for distinct topics
2. Use descriptive names
3. Include relevant participants
4. Archive old conversations
```

---

### Message Management
**Tools:** `mcp_hyperion_chat_api_message_*` (send, get, list, update)

**Purpose:** Send and manage messages in conversations.

**When to use:**
- Sending messages to conversations
- Retrieving message history
- Updating message content
- Managing message metadata

**Best practices:**
```
1. Use appropriate message types
2. Include metadata when relevant
3. Update messages for corrections
4. Maintain conversation context
```

---

### Context Management
**Tools:** `mcp_hyperion_chat_api_context_*`

**Purpose:** Manage chat context and user information.

**When to use:**
- Setting active conversation
- Getting user context
- Clearing context
- Managing session state

---

## 👥 STAFF & AGENT MANAGEMENT

### Agent Management
**Tools:** `mcp_hyperion_staff_api_agent_*` (create, get, list, update, delete, scale, status)

**Purpose:** Create and manage AI agents.

**When to use:**
- Creating new agents for specific roles
- Updating agent configuration
- Scaling agents
- Checking agent status

**Best practices:**
```
1. Create agents with clear roles
2. Provide detailed system prompts
3. Set appropriate replicas for workload
4. Monitor agent status regularly
5. Update prompts based on performance
```

**Example:**
```
mcp_hyperion_staff_api_agent_create(
  name: "Backend-Developer",
  agentRole: "Developer",
  systemPrompt: "You are a Go backend specialist...",
  replicas: 2
)
```

---

### Person Management
**Tools:** `mcp_hyperion_staff_api_person_*` (create, get, list, update)

**Purpose:** Manage human team members.

**When to use:**
- Adding team members
- Updating profiles
- Listing team
- Managing skills

---

### Prompt Management
**Tools:** `mcp_hyperion_staff_api_prompt_*` (create, get, list, update, delete)

**Purpose:** Create and manage system prompts for agents.

**When to use:**
- Creating role-specific prompts
- Updating prompt content
- Managing multiple prompt versions
- Setting active prompts

**Best practices:**
```
1. Create clear, detailed prompts
2. Include examples and patterns
3. Specify expected behavior
4. Test prompts before activation
5. Version prompts for tracking
```

---

## 📊 DATA & ANALYTICS

### Data Storage & Querying
**Tools:** `mcp_hyperion_data_api_*` (store, query, batch_store, delete, stats)

**Purpose:** Store and query structured data.

**When to use:**
- Storing process data
- Querying analytics
- Tracking metrics
- Generating reports

**Best practices:**
```
1. Define schemas before storing
2. Use consistent data types
3. Include metadata
4. Query efficiently
5. Archive old data
```

---

### Schema Management
**Tools:** `mcp_hyperion_data_api_schema_*` (define, get, list, link)

**Purpose:** Define and manage data schemas.

**When to use:**
- Creating data structure definitions
- Validating data
- Linking schemas to processes
- Managing schema versions

---

### Chart & Visualization
**Tools:** `mcp_hyperion_data_api_query_and_chart`, `mcp_hyperion_data_api_chart_create`

**Purpose:** Create data visualizations.

**When to use:**
- Visualizing analytics
- Creating reports
- Displaying metrics
- Sharing insights

**Best practices:**
```
1. Choose appropriate chart type
2. Use clear labels and titles
3. Include data source
4. Make charts interactive when possible
```

---

## 🌐 EXTERNAL INTEGRATIONS

### Facebook Ads Management
**Tools:** `mcp_hyperion_facebook_ads_mcp_*` (campaigns, ad sets, ads, creatives, insights, etc.)

**Purpose:** Manage Facebook advertising campaigns.

**When to use:**
- Creating campaigns
- Managing ad sets
- Creating ads
- Tracking performance
- Monitoring spend

**Best practices:**
```
1. Authenticate first with facebook_ads_login
2. Check budget before creating campaigns
3. Monitor spend regularly
4. Use insights for optimization
5. Test different creatives
```

---

### Google Search
**Tools:** `mcp_hyperion_google_mcp_google_search`

**Purpose:** Search the web using Google Custom Search.

**When to use:**
- Finding information
- Research
- Fact-checking
- Gathering context

**Best practices:**
```
1. Use specific search queries
2. Limit results appropriately
3. Specify language when relevant
4. Verify results from multiple sources
```

---

### Browser Automation
**Tools:** `mcp_hyperion_browser_mcp_*` (create_session, navigate, click, type, screenshot, etc.)

**Purpose:** Automate browser interactions.

**When to use:**
- Testing web applications
- Scraping data
- Automating workflows
- Taking screenshots
- Filling forms

**Best practices:**
```
1. Create session first
2. Wait for page load
3. Use appropriate selectors
4. Handle timeouts gracefully
5. Clean up sessions
```

---

### Logging & Monitoring
**Tools:** `mcp_hyperion_loki_mcp_*` (query, labels, label_values, series)

**Purpose:** Query and analyze logs.

**When to use:**
- Debugging issues
- Monitoring system health
- Analyzing performance
- Tracking errors

**Best practices:**
```
1. Use appropriate time ranges
2. Filter by relevant labels
3. Use LogQL for complex queries
4. Archive logs for compliance
```

---

## 🎯 BEST PRACTICES SUMMARY

### General Principles
1. **Start with discovery** - Use code_index_search before making changes
2. **Async operations** - Always poll status for async tools
3. **Storage management** - Use Hyperion storage for all generated files
4. **Error handling** - Check task status and handle failures gracefully
5. **Documentation** - Keep context and decisions documented

### Workflow Pattern
```
1. Create human task (coordinator_create_human_task)
2. Search code (code_index_search)
3. Analyze complexity (analyze_complexity)
4. Create agent task (create_agent_task)
5. Execute work (read/write files, run tools)
6. Update progress (coordinator_update_todo_status)
7. Complete task (coordinator_update_task_status)
```

### Tool Selection Matrix

| Goal | Primary Tool | Secondary Tools |
|------|--------------|-----------------|
| Find code | code_index_search | code_index_get_full_content |
| Create task | create_agent_task | coordinator_create_human_task |
| Track progress | coordinator_update_todo_status | coordinator_update_task_status |
| Generate video | google_generate_video | hedra_create_video, heygen_create_video |
| Generate image | bfl_text_to_image | google_generate_image, hedra_create_image |
| Process video | ffmpeg_resize_video | ffmpeg_trim_video, ffmpeg_merge_audio_video |
| Analyze content | google_understand_* | ffmpeg_detect_faces, google_detect_text_regions |
| Manage data | data_api_store | data_api_query, data_api_chart_create |
| Automate browser | browser_create_session | browser_navigate, browser_click, browser_type |

---

## 🚀 QUICK START EXAMPLES

### Example 1: Code Modification Task
```
1. coordinator_create_human_task(prompt: "Add error handling to auth module")
2. code_index_search(query: "authentication error handling")
3. analyze_complexity(description: "...", filesModified: [...])
4. create_agent_task(
     agentName: "Developer-Backend",
     role: "Go Backend Development",
     todos: ["Add error types", "Update handlers", "Add tests"],
     contextSummary: "...",
     filesModified: ["./api/auth.go"],
     complexityAnalysis: {...}
   )
5. [Agent executes work]
6. coordinator_update_todo_status(status: "completed")
7. coordinator_update_task_status(status: "completed")
```

### Example 2: Video Generation Task
```
1. coordinator_create_human_task(prompt: "Create product demo video")
2. create_agent_task(
     agentName: "Content-Creator",
     role: "Video Production",
     todos: ["Write script", "Generate video", "Add subtitles"]
   )
3. mcp_hyperion_google_mcp_google_generate_video(
     prompt: "Product demo showing key features",
     duration: 30
   )
4. [Poll status with google_task_get]
5. mcp_hyperion_ffmpeg_mcp_ffmpeg_add_subtitles(
     videoResourcePath: "hyperion://files/video.mp4",
     subtitleResourcePath: "hyperion://files/subs.srt"
   )
6. [Complete task]
```

### Example 3: Data Analysis Task
```
1. coordinator_create_human_task(prompt: "Analyze campaign performance")
2. mcp_hyperion_facebook_ads_mcp_facebook_insights_get(
     objectId: "campaign_id",
     objectLevel: "campaign",
     fields: ["impressions", "clicks", "spend"]
   )
3. mcp_hyperion_data_api_data_store(
     processType: "campaign_analytics",
     data: {...}
   )
4. mcp_hyperion_data_api_query_and_chart(
     processType: "campaign_analytics",
     chartType: "line",
     title: "Campaign Performance"
   )
5. [Share results]
```

---

## 📝 Notes

- Always use **relative paths** from project root (e.g., `./api/auth.go`)
- Use **hyperion://files/** URIs for storage resources
- **ASYNC operations** require polling with appropriate task_get tools
- **Preserve formatting** when editing files
- **Test changes** before marking complete (run linters, type checks)
- **Document decisions** in task notes and comments

---

**Last Updated:** 2024
**Version:** 1.0
