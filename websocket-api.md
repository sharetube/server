# WebSocket API Reference

## Connection
Create room: `/api/v1/room/create/ws?username=&color=&avatar-url=&video-url=`

Join room: `/api/v1/room/{room-id}/join/ws?username=&color=&avatar-url=&auth-token=`

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
</table>

### Server -> Client
<table>
<tr>
    <td>Type</td>
    <td>Data</td>
</tr>

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
<td>VIDEO_ADDED</td>
<td>

```json
{
    "added_video": {
        "id": "string",
        "url": "string",
        "added_by": "string"
    },
    "playlist": [
        {
            "id": "string",
            "url": "string",
            "added_by": "string"
        }
    ]
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