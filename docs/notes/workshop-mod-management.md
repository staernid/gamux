# Workshop & Mod Dependency Management Design Notes

> Note for future grilling & domain modeling session.

## Overview
Currently, `gamux workshop <url_or_id>` performs basic direct downloads into a target folder, but is not yet deeply integrated into the `engine` lifecycle or `GameStatus` inspection.

## Key Design Questions for Grilling Session
1. **Mod Directory Placement & Engine Topologies**:
   - How should `gamux` detect game engine mod directories (e.g. Unity `BepInEx/plugins/`, Unreal `Content/Paks/`, custom `mods/` folders)?
   - Should `engine.ApplyGame` support automated mod structure normalization?
2. **Workshop Update Tracking**:
   - How to track installed item versions/timestamps on disk vs. Steam Workshop API (`ISteamRemoteStorage/GetPublishedFileDetails`)?
   - Should we include outdated workshop mod alerts alongside official depot updates?
3. **GUI Integration**:
   - Visual mod manager tab in Wails desktop app for enabling/disabling, updating, and adding workshop IDs.
