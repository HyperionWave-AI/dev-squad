/**
 * TaskProgressIndicator Component
 *
 * Displays animated progress indicators during task orchestration and execution.
 * Shows step-by-step progress with visual feedback to prevent static screens.
 */

import { Box, Paper, Typography, LinearProgress, Fade, Chip } from '@mui/material';
import {
  SmartToy,
  CheckCircle,
  AutoAwesome,
  Code,
} from '@mui/icons-material';
import { keyframes } from '@mui/system';

// Pulse animation for the icon
const pulse = keyframes`
  0% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.8;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
`;

// Shimmer animation for the progress bar
const shimmer = keyframes`
  0% {
    background-position: -1000px 0;
  }
  100% {
    background-position: 1000px 0;
  }
`;

interface ToolCallInfo {
  id: string;
  tool: string;
  isPending: boolean;
}

interface TaskProgressIndicatorProps {
  mode: 'thinking' | 'working' | 'orchestrating';
  currentStep?: string;
  estimatedTime?: string;
  toolCalls?: ToolCallInfo[];
}

// Map tool names to user-friendly descriptions
const getToolDescription = (toolName: string): string => {
  const toolDescriptions: Record<string, string> = {
    // Coordinator tools
    'coordinator_create_human_task': 'Creating task from your request',
    'coordinator_create_agent_task': 'Setting up specialist agent task',
    'create_agent_task': 'Assigning work to specialist agent',
    'coordinator_update_task_status': 'Updating task progress',
    'coordinator_update_todo_status': 'Marking TODO items as complete',
    'execute_subagent': 'Launching specialist agent',

    // Code search & analysis
    'code_index_search': 'Searching codebase for relevant files',
    'code_index_scan': 'Scanning project structure',
    'code_index_add_folder': 'Indexing code files',

    // Knowledge & learning
    'coordinator_upsert_knowledge': 'Saving learnings to knowledge base',
    'query_knowledge': 'Retrieving relevant knowledge',
    'knowledge_store': 'Storing pattern for future use',
    'knowledge_find': 'Finding similar past solutions',

    // File operations
    'read_file': 'Reading source file',
    'write_file': 'Writing changes to file',
    'list_directory': 'Listing directory contents',
    'apply_patch': 'Applying code changes',

    // Execution
    'bash': 'Running command',

    // Tools discovery
    'discover_tools': 'Finding available tools',
    'get_tool_schema': 'Getting tool details',
    'execute_tool': 'Running specialized tool',
  };

  return toolDescriptions[toolName] || `Executing ${toolName.replace(/_/g, ' ')}`;
};

export function TaskProgressIndicator({
  mode,
  currentStep,
  estimatedTime,
  toolCalls = [],
}: TaskProgressIndicatorProps) {
  // Determine icon and colors based on mode
  const getIconAndColor = () => {
    switch (mode) {
      case 'thinking':
        return {
          icon: <SmartToy sx={{ fontSize: 24 }} />,
          color: 'grey',
          bgColor: 'grey.100',
          primaryColor: 'text.secondary',
        };
      case 'orchestrating':
        return {
          icon: <AutoAwesome sx={{ fontSize: 24 }} />,
          color: 'secondary',
          bgColor: 'secondary.50',
          primaryColor: 'secondary.dark',
        };
      case 'working':
        return {
          icon: <Code sx={{ fontSize: 24 }} />,
          color: 'primary',
          bgColor: 'primary.50',
          primaryColor: 'primary.dark',
        };
    }
  };

  const { icon, color, bgColor, primaryColor } = getIconAndColor();

  // Get current and completed tools
  const pendingTools = toolCalls.filter(tc => tc.isPending);
  const completedTools = toolCalls.filter(tc => !tc.isPending);

  // Get the most recent tool being executed
  const currentTool = pendingTools.length > 0 ? pendingTools[pendingTools.length - 1] : null;
  const currentToolDescription = currentTool ? getToolDescription(currentTool.tool) : null;

  return (
    <Fade in timeout={300}>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'flex-start',
          mb: 2,
          px: 2,
        }}
      >
        <Box
          sx={{
            display: 'flex',
            gap: 1.5,
            alignItems: 'flex-start',
            maxWidth: '75%',
          }}
        >
          {/* Animated Avatar */}
          <Box
            sx={{
              width: 40,
              height: 40,
              borderRadius: '50%',
              backgroundColor: `${color}.light`,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
              animation: `${pulse} 2s ease-in-out infinite`,
              boxShadow: (theme) => {
                const paletteColor = theme.palette[color as 'primary' | 'secondary' | 'grey'];
                const lightColor = typeof paletteColor === 'object' && 'light' in paletteColor ? paletteColor.light : paletteColor;
                return `0 0 20px ${lightColor}`;
              },
            }}
          >
            {icon}
          </Box>

          {/* Progress Content */}
          <Paper
            elevation={2}
            sx={{
              flex: 1,
              px: 2.5,
              py: 2,
              backgroundColor: bgColor,
              borderRadius: 2,
              borderLeft: 4,
              borderColor: `${color}.main`,
              minWidth: 300,
            }}
          >
            {/* Main Status */}
            <Box sx={{ mb: 1.5 }}>
              <Typography
                variant="body1"
                color={primaryColor}
                fontWeight={600}
                sx={{ mb: 0.5 }}
              >
                {mode === 'thinking' && 'AI is thinking...'}
                {mode === 'orchestrating' && 'Orchestrating task...'}
                {mode === 'working' && 'Executing task...'}
              </Typography>

              {/* Show current tool being executed */}
              {currentToolDescription && (
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1, fontWeight: 500 }}>
                  ⚡ {currentToolDescription}
                </Typography>
              )}

              {/* Show static step if no tool info */}
              {!currentToolDescription && currentStep && (
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                  {currentStep}
                </Typography>
              )}

              {/* Animated Progress Bar */}
              <LinearProgress
                sx={{
                  height: 4,
                  borderRadius: 2,
                  backgroundColor: 'rgba(0, 0, 0, 0.08)',
                  '& .MuiLinearProgress-bar': {
                    borderRadius: 2,
                    background: (theme) => {
                      const paletteColor = theme.palette[color as 'primary' | 'secondary' | 'grey'];
                      const lightColor = typeof paletteColor === 'object' && 'light' in paletteColor ? paletteColor.light : paletteColor;
                      const mainColor = typeof paletteColor === 'object' && 'main' in paletteColor ? paletteColor.main : paletteColor;
                      return `linear-gradient(90deg, ${lightColor}, ${mainColor}, ${lightColor})`;
                    },
                    backgroundSize: '1000px 100%',
                    animation: `${shimmer} 2s infinite linear`,
                  },
                }}
              />
            </Box>

            {/* Real-time Tool Execution Display */}
            {toolCalls.length > 0 && (
              <Box sx={{ mt: 1.5 }}>
                {/* Completed tools */}
                {completedTools.length > 0 && (
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, mb: 1 }}>
                    {completedTools.map((toolCall) => (
                      <Chip
                        key={toolCall.id}
                        icon={<CheckCircle sx={{ fontSize: 14 }} />}
                        label={getToolDescription(toolCall.tool)}
                        size="small"
                        sx={{
                          fontSize: '0.7rem',
                          height: 22,
                          backgroundColor: 'success.light',
                          color: 'success.contrastText',
                          opacity: 0.8,
                        }}
                      />
                    ))}
                  </Box>
                )}

                {/* Pending/executing tools */}
                {pendingTools.length > 0 && (
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
                    {pendingTools.map((toolCall, index) => (
                      <Chip
                        key={toolCall.id}
                        icon={<Code sx={{ fontSize: 14 }} />}
                        label={getToolDescription(toolCall.tool)}
                        size="small"
                        sx={{
                          fontSize: '0.7rem',
                          height: 22,
                          backgroundColor: 'primary.light',
                          color: 'primary.contrastText',
                          animation: `${pulse} ${1.5 + index * 0.2}s ease-in-out infinite`,
                        }}
                      />
                    ))}
                  </Box>
                )}

                {/* Progress summary */}
                {toolCalls.length > 0 && (
                  <Typography
                    variant="caption"
                    color="text.disabled"
                    sx={{ display: 'block', mt: 1, fontSize: '0.7rem' }}
                  >
                    {completedTools.length} of {toolCalls.length} steps completed
                  </Typography>
                )}
              </Box>
            )}

            {/* Estimated Time */}
            {estimatedTime && (
              <Typography
                variant="caption"
                color="text.disabled"
                sx={{ display: 'block', mt: 1.5, fontStyle: 'italic' }}
              >
                Estimated time: {estimatedTime}
              </Typography>
            )}

            {/* Subtle hint */}
            <Typography
              variant="caption"
              color="text.disabled"
              sx={{ display: 'block', mt: 0.5 }}
            >
              You can send messages to interrupt anytime
            </Typography>
          </Paper>
        </Box>
      </Box>
    </Fade>
  );
}
