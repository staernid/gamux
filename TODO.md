# TODO — gamux Development Roadmap

Track upcoming features, enhancements, and tasks for **gamux**.

---

## Active & Upcoming Tasks

### Phase 4: Batch Library Processor
- [ ] **4a. Multi-Game Directory Scanner**
  - Add `gamux batch <dir>` command to scan a parent directory containing multiple games
- [ ] **4b. Automated Bulk Pipeline**
  - Auto-detect, apply GBE (loader or portable), and register all discovered games in Lutris/Steam in a single pass

### Phase 5: Game Status & Rollback Manager
- [ ] **5a. Status Inspection**
  - Add `gamux status [path]` command to inspect whether a game folder is *Original*, *Loader-Configured*, or *Portable-Patched*
- [ ] **5b. Rollback Engine**
  - Add `gamux rollback [path]` command to restore `.ORIGINAL` `steam_api` libraries and clean up generated `steam_settings`
