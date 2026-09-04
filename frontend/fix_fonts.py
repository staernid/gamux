import os

mapping = {
    'text-[9px]': 'text-4xs',
    'text-[10px]': 'text-3xs',
    'text-[11px]': 'text-2xs',
    'text-[12px]': 'text-xs'
}

files = [
    "src/components/stage/drawer/DrawerAboutTab.svelte",
    "src/components/stage/drawer/DrawerAchievementsTab.svelte",
    "src/components/stage/drawer/DrawerDLCTab.svelte",
    "src/components/stage/drawer/DrawerLogsTab.svelte",
    "src/components/stage/drawer/DrawerNewsTab.svelte",
    "src/components/stage/drawer/DrawerSavesTab.svelte",
    "src/components/stage/drawer/DrawerResourcesTab.svelte",
    "src/components/stage/drawer/DrawerIntegrityTab.svelte",
    "src/components/views/DownloadsView.svelte",
    "src/components/views/tasks/TaskCard.svelte",
    "src/components/views/tasks/TaskHistorySection.svelte",
    "src/components/views/tasks/TaskQueueSection.svelte",
    "src/components/views/hub/HubFilesTab.svelte",
    "src/components/views/hub/HubCommunityTab.svelte",
    "src/components/views/hub/HubResourcesTab.svelte",
    "src/components/views/settings/GeneralTab.svelte",
    "src/components/views/settings/LogsTab.svelte",
    "src/components/views/settings/CompatibilityTab.svelte",
    "src/components/views/settings/PersonaTab.svelte",
    "src/components/views/settings/CredentialsTab.svelte",
    "src/components/views/settings/LibraryStorageTab.svelte"
]

base_dir = "/home/staernid/projects/gamux/frontend"

for f in files:
    path = os.path.join(base_dir, f)
    if not os.path.exists(path):
        print(f"Not found: {f}")
        continue
    with open(path, 'r') as file:
        content = file.read()
    
    orig_content = content
    changes = 0
    for k, v in mapping.items():
        count = content.count(k)
        if count > 0:
            changes += count
            content = content.replace(k, v)
        
    if content != orig_content:
        with open(path, 'w') as file:
            file.write(content)
        print(f"Updated {f}: {changes} replacements")
    else:
        print(f"No changes in {f}")

