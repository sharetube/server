# WebSocket API Reference

## Connection
Create room: `/api/v1/room/create/ws?username=<required>&color=<required>&avatar-url=<required>&video-url=<required>`

Join room: `/api/v1/room/{room-id}/join/ws?username=<required>&color=<required>&avatar-url=<required>&auth-token=<opt>`

## Messages

### Client -> Server
<table>
<tr>
    <td>Type</td>
    <td>Payload</td>
</tr>

<td>PROMOTE_MEMBER</td>
<td>

```json
{
  "member_id": "string"
}
```
</td>

<tr>
<td>REMOVE_MEMBER</td>
<td>

```json
{
  "member_id": "string"
}
```
</td>

<tr>
<td>ADD_VIDEO</td>
<td>

```json
{
  "video_url": "string"
}
```
</td>

<tr>
<td>REMOVE_VIDEO</td>
<td>

```json
{
  "video_id": "string"
}
```
</td>

<tr>
<td>READY</td>
<td>

```json
null
```
</td>

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
</table>

### Server -> Client
<table>
<tr>
    <td>Type</td>
    <td>Data</td>
</tr>

<tr>
<td>ROOM_STATE</td>
<td>

```json
{
  "room": {
    "room_id": "string",
    "player": {
      "playback_rate": "number",
      "is_playing": "boolean",
      "current_time": "number",
      "updated_at": "number"
    },
    "playlist":{
      "videos": [
        {
          "id": "string",
          "url": "string",
          "added_by": "string"
        }
      ],
      "previous_video_id": {
        "id": "string",
        "url": "string",
        "added_by": "string"
      }
    },
    "members": [
      {
        "id": "string",
        "username": "string",
        "color": "string",
        "avatar_url": "string",
        "is_online": "boolean",
        "is_admin": "boolean",
        "is_muted": "boolean"
      }
    ]
  }
}
```
</td>

<tr>
<td>JOINED_ROOM</td>
<td>

```json
{
  "auth_token": "string",
  "room": {
    "room_id": "string",
    "player": {
      "playback_rate": "number",
      "is_playing": "boolean",
      "current_time": "number",
      "updated_at": "number"
    },
    "playlist":{
      "videos": [
        {
          "id": "string",
          "url": "string",
          "added_by": "string"
        }
      ],
      "previous_video_id": {
        "id": "string",
        "url": "string",
        "added_by": "string"
      }
    },
    "members": [
      {
        "id": "string",
        "username": "string",
        "color": "string",
        "avatar_url": "string",
        "is_online": "boolean",
        "is_admin": "boolean",
        "is_muted": "boolean"
      }
    ]
  }
}
```
</td>

<tr>
<td>PLAYER_UPDATED</td>
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
        "url": "string",
        "added_by": "string"
      }
    ],
    "previous_video_id": {
      "id": "string",
      "url": "string",
      "added_by": "string"
    }
  }
}
```
</td>

<tr>
<td>VIDEO_ADDED</td>
<td>

```json
{
  "added_video": {
    "id": "string",
    "url": "string",
    "added_by": "string"
  },
  "playlist":{
    "videos": [
      {
        "id": "string",
        "url": "string",
        "added_by": "string"
      }
    ],
    "previous_video_id": {
      "id": "string",
      "url": "string",
      "added_by": "string"
    }
  }
}
```
</td>

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
    "is_online": "boolean",
    "is_admin": "boolean",
    "is_muted": "boolean"
  },
  "members": [
    {
      "id": "string",
      "username": "string",
      "color": "string",
      "avatar_url": "string",
      "is_online": "boolean",
      "is_admin": "boolean",
      "is_muted": "boolean"
    }
  ]
}
```
</td>

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
      "is_online": "boolean",
      "is_admin": "boolean",
      "is_muted": "boolean"
    }
  ]
}
```
</td>

<tr>
<td>MEMBER_PROMOTED</td>
<td>

```json
{
  "promoted_member_id": "string"
}
```
</td>
</table>