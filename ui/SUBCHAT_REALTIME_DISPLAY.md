# Real-Time Subchat Display Implementation

## Overview
Enhanced the parallel workflow sidebar to display currently running subchats with real-time progress updates. The sidebar now shows active/running subchats separately from completed ones, with visual indicators and automatic polling for status updates.

## Implementation Date
2025-10-20

## Changes Made

### 1. SubchatList Component (`/ui/src/components/SubchatList.tsx`)

#### Auto-Refresh Polling
- **Feature**: Added automatic polling every 5 seconds to refresh subchat status
- **Implementation**: Used `setInterval` in `useEffect` hook with proper cleanup
- **Behavior**:
  - Initial load shows loading spinner
  - Background refreshes don't show spinner to avoid UI flicker
  - Poll interval cleans up when component unmounts or parentChatId changes

```typescript
useEffect(() => {
  loadSubchats();

  // Set up auto-refresh polling every 5 seconds for real-time updates
  const intervalId = setInterval(() => {
    loadSubchats(true); // Pass true to indicate background refresh
  }, 5000);

  // Clean up interval on unmount or when parentChatId changes
  return () => clearInterval(intervalId);
}, [parentChatId]);
```

#### Separated Display Sections
- **Running Subchats**: Displayed prominently at the top with blue header
- **Completed Subchats**: Displayed below running ones with gray header
- **Counts**: Each section shows count of subchats in that state

### 2. SubchatCard Component (`/ui/src/components/SubchatCard.tsx`)

#### Enhanced Visual Indicators

**New Imports Added:**
- `CircularProgress` - Spinning indicator for running subchats
- `LinearProgress` - Progress bar beneath status
- Status icons: `RunningIcon`, `CompletedIcon`, `ErrorIcon`

**Status Icon Mapping:**
```typescript
const STATUS_ICONS: Record<string, React.ElementType> = {
  active: RunningIcon,
  completed: CompletedIcon,
  failed: ErrorIcon,
};
```

**Visual Enhancements:**
1. **Status Icons**: Each status chip now includes an appropriate icon
2. **Circular Progress**: Running subchats show a small spinning indicator next to status
3. **Linear Progress Bar**: Active subchats display an indeterminate progress bar
4. **Pulse Animation**: Running subchat cards have a subtle pulse border animation

```typescript
// Pulse animation for running subchats
...(isRunning && {
  animation: 'pulse 2s ease-in-out infinite',
  '@keyframes pulse': {
    '0%, 100%': { borderColor: 'divider' },
    '50%': { borderColor: 'primary.main', borderWidth: 2 },
  },
})
```

## User Experience

### Before
- Single list of all subchats mixed together
- No real-time updates - required manual refresh
- No visual indication of running vs completed status
- Static display with no progress indicators

### After
- **Running subchats** displayed prominently at top with:
  - Blue "Running" section header with count
  - Pulsing border animation on cards
  - Status badge with running icon and circular spinner
  - Linear progress bar showing activity
  - Auto-refreshes every 5 seconds

- **Completed subchats** in separate section below:
  - Gray "Completed" section header with count
  - Checkmark or error icon based on status
  - No animations or progress indicators

## Technical Details

### Polling Strategy
- **Interval**: 5 seconds (configurable by changing interval value)
- **Error Handling**: Errors during background refresh are captured but don't interrupt polling
- **Cleanup**: Interval properly cleared on component unmount
- **Performance**: Background refreshes skip loading state to prevent UI flicker

### State Management
- Uses existing React `useState` hooks
- No additional state management libraries required
- Leverages existing `subchatService` API calls

### API Integration
- Uses existing `/api/v1/chats/:chatId/subchats` endpoint
- No backend changes required
- Polling interval can be adjusted based on backend performance

## Future Enhancements

### Potential WebSocket Integration
Currently using polling for simplicity. For even more real-time updates, consider:

```typescript
// Future WebSocket implementation
useEffect(() => {
  const ws = new WebSocket(`ws://host/api/v1/subchats/${parentChatId}/stream`);

  ws.onmessage = (event) => {
    const updatedSubchat = JSON.parse(event.data);
    setSubchats(prev =>
      prev.map(s => s.id === updatedSubchat.id ? updatedSubchat : s)
    );
  };

  return () => ws.close();
}, [parentChatId]);
```

### Progress Percentage
If backend provides completion percentage, enhance the progress bar:

```typescript
{isRunning && subchat.progress !== undefined && (
  <LinearProgress
    variant="determinate"
    value={subchat.progress}
    color="primary"
  />
)}
```

### Message Count Badge
Show number of messages in each subchat:

```typescript
<Badge badgeContent={subchat.messageCount} color="primary">
  <SubchatCard subchat={subchat} />
</Badge>
```

## Testing Recommendations

### Manual Testing
1. Create multiple subchats with different agents
2. Verify running subchats appear in "Running" section
3. Watch for auto-refresh every 5 seconds (check network tab)
4. Complete a subchat and verify it moves to "Completed" section
5. Verify pulse animation on running subchat cards
6. Verify progress indicators appear only on running subchats

### Automated Testing
```typescript
// Example test for SubchatList
it('should auto-refresh subchats every 5 seconds', async () => {
  jest.useFakeTimers();
  const mockGetSubchats = jest.fn();

  render(<SubchatList parentChatId="123" />);

  expect(mockGetSubchats).toHaveBeenCalledTimes(1);

  jest.advanceTimersByTime(5000);
  expect(mockGetSubchats).toHaveBeenCalledTimes(2);

  jest.advanceTimersByTime(5000);
  expect(mockGetSubchats).toHaveBeenCalledTimes(3);

  jest.useRealTimers();
});
```

## Performance Considerations

### Polling Overhead
- API call every 5 seconds per open sidebar
- Consider increasing interval if backend load is high
- Background refreshes don't block UI

### Animation Performance
- CSS animations use GPU acceleration (transform, opacity)
- Border pulse animation is lightweight
- No JavaScript-based animations that could cause jank

### Memory Management
- Interval properly cleaned up on unmount
- No memory leaks from orphaned timers
- React state updates are batched

## Files Modified

1. `/ui/src/components/SubchatList.tsx`
   - Added auto-refresh polling with 5-second interval
   - Separated running and completed subchats into sections
   - Enhanced section headers with counts and colors

2. `/ui/src/components/SubchatCard.tsx`
   - Added status icons (RunningIcon, CompletedIcon, ErrorIcon)
   - Added CircularProgress spinner for running subchats
   - Added LinearProgress bar for running subchats
   - Added pulse border animation for running subchat cards
   - Enhanced status chip with icon display

## No Backend Changes Required
This implementation uses existing API endpoints and doesn't require any backend modifications.

## Accessibility Improvements
- Status icons provide visual cues beyond color
- ARIA labels maintained from original implementation
- Progress indicators use semantic MUI components with built-in ARIA support

## Browser Compatibility
- Uses standard JavaScript `setInterval` (universal support)
- MUI components handle browser differences
- CSS animations use standard keyframe syntax
- No experimental features used

## Configuration

To adjust polling interval, modify the interval value in `SubchatList.tsx`:

```typescript
const intervalId = setInterval(() => {
  loadSubchats(true);
}, 5000); // Change this value (in milliseconds)
```

Recommended values:
- **3000ms (3s)**: High-frequency updates, more server load
- **5000ms (5s)**: Balanced (current setting)
- **10000ms (10s)**: Low-frequency, minimal server load
