import os

mapping = {
    'text-[9px]': 'text-4xs',
    'text-[10px]': 'text-3xs',
    'text-[11px]': 'text-2xs',
    'text-[12px]': 'text-xs'
}

files = [
    "src/components/stage/metadata/MetadataTab.svelte",
    "src/components/views/SettingsView.svelte"
]

base_dir = "/home/staernid/projects/gamux/frontend"

for f in files:
    path = os.path.join(base_dir, f)
    if not os.path.exists(path):
        continue
    with open(path, 'r') as file:
        content = file.read()
    
    orig_content = content
    for k, v in mapping.items():
        content = content.replace(k, v)
        
    if content != orig_content:
        with open(path, 'w') as file:
            file.write(content)
        print(f"Updated {f}")

