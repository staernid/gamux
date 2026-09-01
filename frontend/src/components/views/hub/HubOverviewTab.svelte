<script>
  import DrawerAboutTab from '../../stage/drawer/DrawerAboutTab.svelte';
  import DrawerResourcesTab from '../../stage/drawer/DrawerResourcesTab.svelte';

  export let game = null;
</script>

<div class="space-y-6">
  <!-- Game Summary Synopsis & Facts -->
  <DrawerAboutTab {game} />

  <!-- Categorized Technical & Community Resources Grid -->
  <DrawerResourcesTab {game} />
</div>
