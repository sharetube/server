# WebSocket API Reference

## Connection
Create room: `/api/v1/room/create/ws?username=<required>&color=<required>&avatar-url=<optional>&video-url=<required>`

Join room: `/api/v1/room/{room-id}/join/ws?jwt=<optional>&username=<required>&color=<required>&avatar-url=<optional>`

## Message base structure
```json
{
  "type": "string",
  "payload": "object"
}
```

## Messages

### Client -> Server
<table>
<tr>
    <td>Type</td>
    <td>Payload</td>
</tr>

<tr>
<td>UPDATE_PROFILE</td>
<td>

```json
{
  "username": "string",
  "color": "string",
  "avatar_url": "string",
}
```
</td>
</tr>

<tr>
<td>PROMOTE_MEMBER</td>
<td>

```json
{
  "member_id": "string"
}
```
</td>
</tr>

<tr>
<td>REMOVE_MEMBER</td>
<td>

```json
{
  "member_id": "string"
}
```
</td>
</tr>

<tr>
<td>ADD_VIDEO</td>
<td>

```json
{
  "video_url": "string"
}
```
</td>
</tr>

<tr>
<td>REMOVE_VIDEO</td>
<td>

```json
{
  "video_id": "string"
}
```
</td>
</tr>

<tr>
<td>REORDER_PLAYLIST</td>
<td>

```json
{
  "videos": [
    {
      "id": "string",
      "url": "string"
    }
  ],
}
```
</td>
</tr>

<tr>
<td>UPDATE_READY</td>
<td>

```json
{
  "is_ready": "boolean"
}
```
</td>
</tr>

<tr>
<td>UPDATE_MUTED</td>
<td>

```json
{
  "is_muted": "boolean"
}
```
</td>
</tr>

<tr>
<td>UPDATE_PLAYER_STATE</td>
<td>

```json
{
  "playback_rate": "number",
  "is_playing": "boolean",
  "current_time": "number",
  "updated_at": "number"
}
```
</td>
</tr>

<tr>
<td>UPDATE_PLAYER_VIDEO</td>
<td>

```json
{
  "video_id": "string",
  "updated_at": "number"
}
```
</td>
</tr>
</table>

### Server -> Client
<table>
<tr>
    <td>Type</td>
    <td>Payload</td>
</tr>

<tr>
<td>JOINED_ROOM</td>
<td>

```json
{
  "jwt": "string",
  "joined_member": {
    "id": "string",
    "username": "string",
    "color": "string",
    "avatar_url": "string",
    "is_ready": "boolean",
    "is_admin": "boolean",
    "is_muted": "boolean"
  },
  "room": {
    "room_id": "string",
    "player": {
      "video_url": "string",
      "playback_rate": "number",
      "is_playing": "boolean",
      "current_time": "number",
      "updated_at": "number"
    },
    "playlist":{
      "videos": [
        {
          "id": "string",
          "url": "string"
        }
      ],
      "last_video_id": {
        "id": "string",
        "url": "string"
      }
    },
    "members": [
      {
        "id": "string",
        "username": "string",
        "color": "string",
        "avatar_url": "string",
        "is_ready": "boolean",
        "is_admin": "boolean",
        "is_muted": "boolean"
      }
    ]
  }
}
```
</td>
</tr>

<tr>
<td>PLAYER_STATE_UPDATED</td>
<td>

```json
{
  "player": {
    "playback_rate": "number",
    "is_playing": "boolean",
    "current_time": "number",
    "updated_at": "number"
  }
}
```
</td>
</tr>

<tr>
<td>PLAYER_VIDEO_UPDATED</td>
<td>

```json
{
  "player": {
    "video_url": "string",
    "playback_rate": "number",
    "is_playing": "boolean",
    "current_time": "number",
    "updated_at": "number"
  },
  "playlist":{
    "videos": [
      {
        "id": "string",
        "url": "string"
      }
    ],
    "last_video_id": {
      "id": "string",
      "url": "string"
    }
  }
}
```
</td>
</tr>

<tr>
<td>VIDEO_ADDED</td>
<td>

```json
{
  "added_video": {
    "id": "string",
    "url": "string"
  },
  "playlist":{
    "videos": [
      {
        "id": "string",
        "url": "string"
      }
    ],
    "last_video_id": {
      "id": "string",
      "url": "string"
    }
  }
}
```
</td>
</tr>

<tr>
<td>VIDEO_REMOVED</td>
<td>

```json
{
  "removed_video_id": "string",
  "playlist":{
    "videos": [
      {
        "id": "string",
        "url": "string"
      }
    ],
    "last_video_id": {
      "id": "string",
      "url": "string"
    }
  }
}
```
</td>
</tr>

<tr>
<td>PLAYLIST_REORDERED</td>
<td>

```json
{
  "videos": [
    {
      "id": "string",
      "url": "string"
    }
  ],
  "last_video_id": {
    "id": "string",
    "url": "string"
  }
}
```
</td>
</tr>

<tr>
<td>MEMBER_JOINED</td>
<td>

```json
{
  "joined_member": {
    "id": "string",
    "username": "string",
    "color": "string",
    "avatar_url": "string",
    "is_ready": "boolean",
    "is_admin": "boolean",
    "is_muted": "boolean"
  },
  "members": [
    {
      "id": "string",
      "username": "string",
      "color": "string",
      "avatar_url": "string",
      "is_ready": "boolean",
      "is_admin": "boolean",
      "is_muted": "boolean"
    }
  ]
}
```
</td>
</tr>

<tr>
<td>MEMBER_DISCONNECTED</td>
<td>

```json
{
  "disconnected_member_id": "string",
  "members": [
    {
      "id": "string",
      "username": "string",
      "color": "string",
      "avatar_url": "string",
      "is_ready": "boolean",
      "is_admin": "boolean",
      "is_muted": "boolean"
    }
  ]
}
```
</td>
</tr>

<tr>
<td>MEMBER_UPDATED</td>
<td>

```json
{
  "updated_member": {
    "id": "string",
    "username": "string",
    "color": "string",
    "avatar_url": "string",
    "is_ready": "boolean",
    "is_admin": "boolean",
    "is_muted": "boolean"
  },
  "members": [
    {
      "id": "string",
      "username": "string",
      "color": "string",
      "avatar_url": "string",
      "is_ready": "boolean",
      "is_admin": "boolean",
      "is_muted": "boolean"
    }
  ]
}
```
</td>
</tr>

<tr>
<td>IS_ADMIN_CHANGED</td>
<td>

```json
{
  "is_admin": "boolean"
}
```
</td>
</tr>
</table>