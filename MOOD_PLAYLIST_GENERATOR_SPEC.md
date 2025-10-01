# Mood Playlist Generator - Application Specification

**Version:** 1.0
**Date:** 2025-10-01
**Type:** Go + React Web Application

---

## Overview

A simple web application where users describe their current mood or situation, and the app generates a custom playlist with a name, description, cover art concept, and song recommendations. The experience is fast, visual, and delightful.

---

## User Experience

### Flow

1. **Landing Page**
   - Clean, minimal design
   - Single text input: "How are you feeling right now?"
   - Placeholder examples: "Rainy Sunday afternoon", "Ready to crush my workout", "Heartbroken and eating ice cream"
   - Submit button

2. **Generation Screen**
   - Smooth transition from landing
   - Animated gradient background that shifts colors
   - Progress indicators showing generation steps
   - Text updates: "Analyzing your vibe...", "Crafting playlist...", "Selecting songs..."
   - Takes 5-10 seconds total

3. **Results Page**
   - Beautiful full-screen gradient background (matches mood)
   - Playlist name displayed prominently
   - Short vibe description (2-3 sentences)
   - Cover art description (visual concept)
   - 10-12 song recommendations with artist names
   - Actions: "Generate Another", "Share" (copy link)

---

## Technical Architecture

### Frontend (React + TypeScript + Vite)

**Pages:**
```
/                 - Landing page with mood input
/playlist/:id     - Generated playlist display
```

**Components:**
- `MoodInput` - Text input with submit
- `GenerationLoader` - Animated loading with progress text
- `PlaylistDisplay` - Results page with gradient
- `SongList` - List of songs with formatting
- `ShareButton` - Copy playlist link

**State Management:**
- React hooks (useState, useEffect)
- No external state library needed

**Styling:**
- Tailwind CSS for utility classes
- Custom gradients based on mood tone
- Smooth animations (fade, slide)

**API Calls:**
```typescript
POST /api/playlists
{
  "mood": "feeling energized and ready to workout"
}

Response:
{
  "id": "abc123",
  "name": "Power Hour",
  "description": "High-energy beats to fuel your workout...",
  "gradient": ["#FF6B6B", "#4ECDC4"],
  "coverArt": "Abstract geometric shapes in vibrant oranges and blues",
  "songs": [
    { "title": "Eye of the Tiger", "artist": "Survivor" },
    ...
  ],
  "createdAt": "2025-10-01T12:00:00Z"
}
```

---

### Backend (Go + Gin)

**API Endpoints:**

```go
POST /api/playlists
- Input: JSON with "mood" string
- Process: Generate playlist using AI
- Output: Complete playlist object

GET /api/playlists/:id
- Input: Playlist ID
- Process: Retrieve from storage
- Output: Playlist object

GET /api/playlists/recent
- Input: None
- Process: Get last 10 playlists
- Output: Array of playlist objects (optional feature)
```

**Data Models:**

```go
type Playlist struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Gradient    []string  `json:"gradient"`    // 2 hex colors
    CoverArt    string    `json:"coverArt"`    // Text description
    Songs       []Song    `json:"songs"`
    CreatedAt   time.Time `json:"createdAt"`
}

type Song struct {
    Title  string `json:"title"`
    Artist string `json:"artist"`
}
```

**Storage:**
- In-memory map for MVP (id → Playlist)
- Optional: SQLite for persistence
- Optional: Redis for caching

**AI Integration:**
- Use Claude/GPT API to generate playlist
- Single prompt that returns structured JSON
- Prompt engineering to ensure consistent format

**Example Prompt:**
```
User mood: "Rainy Sunday afternoon, cozy vibes"

Generate a playlist with:
1. Playlist name (3-5 words, creative)
2. Description (2-3 sentences about the vibe)
3. Color gradient (2 hex colors that match the mood)
4. Cover art concept (1 sentence visual description)
5. 12 song recommendations (title + artist)

Return as JSON with this structure:
{
  "name": "...",
  "description": "...",
  "gradient": ["#...", "#..."],
  "coverArt": "...",
  "songs": [{"title": "...", "artist": "..."}, ...]
}
```

---

## UI Design

### Color Schemes by Mood

```
Energetic:  #FF6B6B → #FFA500 (red to orange)
Calm:       #A8E6CF → #56CCF2 (mint to sky blue)
Melancholy: #6B5B95 → #4A4A4A (purple to gray)
Happy:      #FFD93D → #FF6BCB (yellow to pink)
Romantic:   #FF6F91 → #C44569 (pink to rose)
```

### Typography

- **Heading:** System font, 48px, bold
- **Body:** System font, 18px, regular
- **Songs:** Monospace, 14px

### Animations

- **Landing → Generation:** Fade out, slide up
- **Generation → Results:** Cross-fade with gradient morph
- **Song List:** Stagger fade-in (each song 100ms delay)

---

## Implementation Plan

### Phase 1: MVP (3-4 hours)

**Backend:**
1. Set up Go + Gin project
2. Create `/api/playlists` POST endpoint
3. Integrate Claude API for generation
4. In-memory storage
5. CORS enabled for local dev

**Frontend:**
1. Set up React + Vite + Tailwind
2. Create `MoodInput` component
3. Create `PlaylistDisplay` component
4. API integration
5. Basic gradient backgrounds

**Testing:**
- Generate 5 different mood playlists
- Verify JSON structure
- Test gradient transitions

---

### Phase 2: Polish (1-2 hours)

1. Add loading animations
2. Improve gradient selection logic
3. Add "Share" functionality (copy URL)
4. Error handling (API failures, network issues)
5. Mobile responsive design

---

### Phase 3: Optional Enhancements

- **Persistence:** Save to SQLite
- **Gallery:** Show recent playlists on homepage
- **Export:** Download as Spotify playlist (requires auth)
- **Voting:** Upvote/downvote playlists
- **Surprise Me:** Random mood suggestion button

---

## File Structure

```
mood-playlist-generator/
├── backend/
│   ├── main.go
│   ├── handlers/
│   │   └── playlist.go
│   ├── models/
│   │   └── playlist.go
│   ├── services/
│   │   └── ai_generator.go
│   ├── storage/
│   │   └── memory.go
│   └── go.mod
│
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── MoodInput.tsx
│   │   │   ├── GenerationLoader.tsx
│   │   │   ├── PlaylistDisplay.tsx
│   │   │   └── SongList.tsx
│   │   ├── services/
│   │   │   └── api.ts
│   │   ├── types/
│   │   │   └── playlist.ts
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   └── vite.config.ts
│
└── README.md
```

---

## Environment Variables

**Backend:**
```bash
# .env
PORT=8080
CLAUDE_API_KEY=sk-ant-...
CORS_ORIGIN=http://localhost:5173
```

**Frontend:**
```bash
# .env
VITE_API_URL=http://localhost:8080
```

---

## Success Criteria

✅ User can input any mood text
✅ Playlist generates in < 10 seconds
✅ Results display with beautiful gradient
✅ Songs are relevant to the mood
✅ Gradient colors match the vibe
✅ Shareable via URL
✅ Works on mobile and desktop

---

## Example Outputs

### Input: "Sunday morning, coffee in hand, reading a book"

**Output:**
```json
{
  "name": "Quiet Pages",
  "description": "Gentle acoustic melodies and soft jazz to accompany your peaceful reading session. Perfect background for lazy Sunday mornings.",
  "gradient": ["#D4A574", "#8B7355"],
  "coverArt": "A worn book on a wooden table with steam rising from a ceramic mug",
  "songs": [
    { "title": "Flightless Bird, American Mouth", "artist": "Iron & Wine" },
    { "title": "Holocene", "artist": "Bon Iver" },
    { "title": "Such Great Heights", "artist": "The Postal Service" },
    { "title": "Skinny Love", "artist": "Bon Iver" },
    { "title": "The Moon Song", "artist": "Karen O" },
    { "title": "To Build a Home", "artist": "The Cinematic Orchestra" },
    { "title": "Mad World", "artist": "Gary Jules" },
    { "title": "Falling Slowly", "artist": "Glen Hansard" },
    { "title": "First Day of My Life", "artist": "Bright Eyes" },
    { "title": "Lua", "artist": "Bright Eyes" }
  ]
}
```

---

## Non-Goals (Out of Scope for MVP)

❌ User authentication
❌ Actual music playback
❌ Spotify integration
❌ Social features (comments, likes)
❌ Advanced analytics
❌ Backend persistence (use in-memory)

---

**Ready to build!** This spec should provide everything needed to implement a delightful mood playlist generator.
