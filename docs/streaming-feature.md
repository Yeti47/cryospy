# Video Streaming Feature

## Overview

The video streaming feature allows users to view live video feeds from security cameras through the CryoSpy dashboard. The implementation uses HLS (HTTP Live Streaming) to provide low-latency streaming of recorded clips.

## Flow

1. **Stream Selection** (`/stream`)
   - User navigates to the stream selection page
   - Selects a target client from the dropdown
   - Optionally sets a reference point in time
   - Clicks "Start Streaming" to proceed

2. **Live Streaming** (`/stream/{clientId}`)
   - User is presented with an HLS video player
   - Video player loads segments from the streaming service
   - Playback controls allow play/pause functionality

## API Endpoints

### Stream Selection
- `GET /stream` - Shows the client selection page

### Live Streaming  
- `GET /stream/{clientId}` - Shows the streaming page for a specific client
- `GET /stream/{clientId}/playlist.m3u8?startTime={startTime}&refTime={refTime}` - Returns the HLS playlist
- `GET /stream/{clientId}/segments/{clipId}` - Returns a video segment (.ts file)

## Architecture

### Backend Components

1. **StreamHandler** - HTTP handlers for streaming endpoints
2. **StreamingService** - Core business logic for generating playlists and serving segments
3. **CachedClipNormalizer** - Caches normalized video segments for improved performance
4. **PlaylistGenerator** - Generates M3U8 playlists for HLS streaming
5. **ClipNormalizer** - Converts video clips to HLS-compatible format using FFmpeg

### Frontend Components

1. **Stream Selection Page** - Client selection and configuration
2. **Live Stream Page** - Video player with HLS.js integration
3. **CSS Styling** - Dark theme styling matching the CryoSpy design

## Configuration

Streaming settings can be configured in the main configuration file:

```json
{
  "streaming_settings": {
    "cache": {
      "enabled": true,
      "max_size_bytes": 104857600
    },
    "look_ahead": 10,
    "width": 854,
    "height": 480,
    "video_bitrate": "1000k",
    "video_codec": "libx264",
    "frame_rate": 25
  }
}
```

### Settings

- `cache.enabled` - Whether to enable caching of normalized video segments
- `cache.max_size_bytes` - Maximum cache size in bytes (default: 100MB)
- `look_ahead` - Number of clips to look ahead for streaming (default: 10)
- `width` - Target video width in pixels (default: 854 for 480p 16:9)
- `height` - Target video height in pixels (default: 480 for 480p)
- `video_bitrate` - Target video bitrate for transcoding (default: "1000k")
- `video_codec` - Video codec to use for transcoding (default: "libx264")
- `frame_rate` - Target frame rate for transcoded video (default: 25 fps)

## Dependencies

- **HLS.js** - Client-side HLS player library (loaded from CDN)
- **FFmpeg** - Video transcoding (via goffmpeg library)
- **Gin** - HTTP routing framework
- **Core CryoSpy modules** - Encryption, video management, client management

## Notes

- Video segments are normalized according to the configured resolution, bitrate, and codec settings
- The default settings provide 480p resolution at 1Mbps for optimal streaming performance
- Video codec can be customized based on system availability (e.g., "libx264", "libopenh264", "libx265", "h264_nvenc")
- Segments are cached to improve response times for frequently accessed content
- The streaming service supports both live streaming (new clips as they arrive) and historical playback
- All video data remains encrypted at rest and is only decrypted during streaming
- CORS headers and appropriate cache control headers are set for optimal browser compatibility
