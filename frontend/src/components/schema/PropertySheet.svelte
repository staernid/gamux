<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { AppAPI } from '../../api/app.js';
  import PropertyRow from './PropertyRow.svelte';
  import SwitchControl from './controls/SwitchControl.svelte';
  import SelectControl from './controls/SelectControl.svelte';
  import PathPickerControl from './controls/PathPickerControl.svelte';
  import EnvEditorControl from './controls/EnvEditorControl.svelte';
  import SliderControl from './controls/SliderControl.svelte';
  import TextControl from './controls/TextControl.svelte';
  import { snakeToCamelKey } from '../../stores/drawerDraft.js';
  import { MOCK_CONFIG_SCHEMA } from '../../api/mockData.js';

  export let schema = [];
  export let values = {};
  export let inheritedValues = {}; // Global configuration for inheritance resolution
  export let category = null; // optional category filter: string | string[]
  export let excludeKeys = []; // optional keys to exclude
  export let includeKeys = []; // optional keys to exclusively include
  export let scope = null; // optional scope filter: 'global' | 'game'
  export let groupByCategory = true;
  export let disabled = false;
  export let onChange = null;
  export let onchange = null;
  export let onReset = null;

  const dispatch = createEventDispatcher();

  let loadedSchema = Array.isArray(MOCK_CONFIG_SCHEMA) ? MOCK_CONFIG_SCHEMA : [];

  $: effectiveSchema = (schema && schema.length > 0) ? schema : loadedSchema;

  // Filter schema by scope ('global', 'game', or 'both')
  $: scopedSchema = scope
    ? effectiveSchema.filter(p => {
        if (!p.scope) return true;
        const s = p.scope.toLowerCase();
        const target = scope.toLowerCase();
        return s === target || s === 'both';
      })
    : effectiveSchema;

  // Filter schema by category, includeKeys, and excludeKeys
  $: filteredSchema = scopedSchema.filter(p => {
    if (Array.isArray(includeKeys) && includeKeys.length > 0) {
      if (!includeKeys.includes(p.key)) return false;
    }
    if (Array.isArray(excludeKeys) && excludeKeys.length > 0) {
      if (excludeKeys.includes(p.key)) return false;
    }
    if (category) {
      const pCat = (p.category || '').toLowerCase();
      if (Array.isArray(category)) {
        if (!category.map(c => (c || '').toLowerCase()).includes(pCat)) return false;
      } else {
        if (pCat !== category.toLowerCase()) return false;
      }
    }
    return true;
  });

  // Group by category if groupByCategory is true
  $: categories = groupByCategory
    ? Array.from(new Set(filteredSchema.map(p => p.category || 'General')))
    : [];

  onMount(async () => {
    if (!schema || schema.length === 0) {
      try {
        const res = scope === 'game'
          ? await AppAPI.getGameConfigSchema()
          : await AppAPI.getConfigSchema();
        if (Array.isArray(res)) {
          loadedSchema = res;
        }
      } catch {
        // Fallback / offline handled
      }
    }
  });

  function getNestedVal(obj, key) {
    if (!obj) return undefined;
    if (key in obj) return obj[key];
    const camel = snakeToCamelKey(key);
    if (camel && camel in obj) return obj[camel];
    if (key.includes('.')) {
      const parts = key.split('.');
      let cur = obj;
      for (const part of parts) {
        if (cur == null || typeof cur !== 'object') return undefined;
        if (part in cur) {
          cur = cur[part];
        } else {
          const partCamel = snakeToCamelKey(part);
          cur = (partCamel && partCamel in cur) ? cur[partCamel] : undefined;
        }
      }
      return cur;
    }
    return undefined;
  }

  function resolveInheritedVal(prop, inhValues = inheritedValues) {
    const inheritKey = prop.inherits_from || prop.inheritsFrom || prop.key;
    const globalVal = getNestedVal(inhValues, inheritKey);
    if (globalVal !== undefined && globalVal !== null) {
      return globalVal;
    }
    return prop.default;
  }

  function isPropOverridden(prop, curValues = values, inhValues = inheritedValues) {
    if (scope !== 'game') return false;
    const valInProfile = getNestedVal(curValues, prop.key);
    if (valInProfile === undefined || valInProfile === null || valInProfile === '') {
      return false;
    }
    const inheritedVal = resolveInheritedVal(prop, inhValues);
    const ctrlType = resolveControlType(prop.type);
    if (ctrlType === 'switch') {
      return Boolean(valInProfile) !== Boolean(inheritedVal);
    }
    if (ctrlType === 'slider') {
      return Number(valInProfile) !== Number(inheritedVal);
    }
    if (typeof valInProfile === 'string') {
      return valInProfile.trim() !== String(inheritedVal ?? '').trim();
    }
    if (typeof valInProfile === 'object') {
      return JSON.stringify(valInProfile) !== JSON.stringify(inheritedVal ?? {});
    }
    return valInProfile !== inheritedVal;
  }

  function resolveCurrentVal(prop, curValues = values, inhValues = inheritedValues) {
    const valInProfile = getNestedVal(curValues, prop.key);
    if (valInProfile !== undefined && valInProfile !== null && valInProfile !== '') {
      return valInProfile;
    }
    if (scope === 'game') {
      return resolveInheritedVal(prop, inhValues);
    }
    return prop.default;
  }

  function updatePropertyValue(key, nextVal) {
    if (disabled) return;
    const updatedValues = { ...values };
    const camel = snakeToCamelKey(key);

    if (key.includes('.')) {
      const parts = key.split('.');
      let cur = updatedValues;
      for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i];
        if (!cur[part] || typeof cur[part] !== 'object') {
          cur[part] = { ...cur[part] };
        } else {
          cur[part] = { ...cur[part] };
        }
        cur = cur[part];
      }
      cur[parts[parts.length - 1]] = nextVal;
    } else {
      if (camel && camel in updatedValues && !(key in updatedValues)) {
        updatedValues[camel] = nextVal;
      } else {
        updatedValues[key] = nextVal;
      }
    }

    values = updatedValues;
    const changeDetail = { key, value: nextVal, values: updatedValues };
    dispatch('change', changeDetail);
    if (typeof onChange === 'function') onChange(changeDetail);
    if (typeof onchange === 'function') onchange(changeDetail);
  }

  function handleResetProperty(key) {
    if (disabled) return;
    const updatedValues = { ...values };
    const camel = snakeToCamelKey(key);
    if (key.includes('.')) {
      const parts = key.split('.');
      let cur = updatedValues;
      for (let i = 0; i < parts.length - 1; i++) {
        if (!cur[parts[i]]) return;
        cur = cur[parts[i]];
      }
      delete cur[parts[parts.length - 1]];
    } else {
      delete updatedValues[key];
      if (camel) delete updatedValues[camel];
      if (['mangoHud', 'disableOverlay', 'offlineMode', 'gamescope'].includes(camel)) {
        updatedValues[camel] = null;
      }
    }
    values = updatedValues;
    const resetDetail = { key, values: updatedValues };
    dispatch('reset', resetDetail);
    if (typeof onReset === 'function') onReset(resetDetail);
  }

  function resolveControlType(type) {
    switch ((type || '').toLowerCase()) {
      case 'boolean':
      case 'bool':
      case 'switch':
        return 'switch';
      case 'select':
      case 'enum':
        return 'select';
      case 'path':
      case 'path_list':
      case 'dirpath':
      case 'filepath':
        return 'path';
      case 'env':
      case 'map':
        return 'env';
      case 'number':
      case 'int':
      case 'integer':
      case 'slider':
        return 'slider';
      default:
        return 'text';
    }
  }
</script>

<div class="space-y-6" data-testid="property-sheet">
  {#if filteredSchema.length === 0}
    <div class="p-6 text-center rounded-xl bg-surface-card/20 border border-surface-border">
      <p class="text-xs text-neutral-500 font-mono">No configuration properties match the current criteria.</p>
    </div>
  {:else if groupByCategory}
    {#each categories as cat}
      {@const catProps = filteredSchema.filter(p => (p.category || 'General') === cat)}
      {#if catProps.length > 0}
        <div class="space-y-3" data-testid={`category-section-${cat.toLowerCase().replace(/[^a-z0-9]/g, '-')}`}>
          <div class="flex items-center space-x-2 pb-1 border-b border-surface-border/60">
            <h3 class="text-xs font-bold text-neutral-300 uppercase tracking-wider font-mono">
              {cat}
            </h3>
            <span class="text-3xs font-mono text-neutral-500">
              ({catProps.length})
            </span>
          </div>

          <div class="space-y-2">
            {#each catProps as prop (prop.key)}
              {@const controlType = resolveControlType(prop.type)}
              {@const currentVal = resolveCurrentVal(prop, values, inheritedValues)}
              {@const isOverridden = isPropOverridden(prop, values, inheritedValues)}
              {@const inheritedVal = resolveInheritedVal(prop, inheritedValues)}

              <PropertyRow
                property={prop}
                {isOverridden}
                inheritedValue={inheritedVal}
                scope={scope || 'global'}
                {disabled}
                on:reset={() => handleResetProperty(prop.key)}
              >
                {#if controlType === 'switch'}
                  <SwitchControl
                    property={prop}
                    value={currentVal}
                    {disabled}
                    hideHeader={true}
                    on:change={(e) => updatePropertyValue(prop.key, e.detail)}
                  />
                {:else if controlType === 'select'}
                  <SelectControl
                    property={prop}
                    value={currentVal}
                    {disabled}
                    hideHeader={true}
                    on:change={(e) => updatePropertyValue(prop.key, e.detail)}
                  />
                {:else if controlType === 'path'}
                  <PathPickerControl
                    property={prop}
                    value={currentVal}
                    {disabled}
                    hideHeader={true}
                    on:change={(e) => updatePropertyValue(prop.key, e.detail)}
                  />
                {:else if controlType === 'env'}
                  <EnvEditorControl
                    property={prop}
                    value={currentVal}
                    {disabled}
                    hideHeader={true}
                    on:change={(e) => updatePropertyValue(prop.key, e.detail)}
                  />
                {:else if controlType === 'slider'}
                  <SliderControl
                    property={prop}
                    value={currentVal}
                    {disabled}
                    on:change={(e) => updatePropertyValue(prop.key, e.detail)}
                  />
                {:else}
                  <TextControl
                    property={prop}
                    value={currentVal}
                    {disabled}
                    on:change={(e) => updatePropertyValue(prop.key, e.detail)}
                  />
                {/if}
              </PropertyRow>
            {/each}
          </div>
        </div>
      {/if}
    {/each}
  {:else}
    <div class="space-y-2">
      {#each filteredSchema as prop (prop.key)}
        {@const controlType = resolveControlType(prop.type)}
        {@const currentVal = resolveCurrentVal(prop, values, inheritedValues)}
        {@const isOverridden = isPropOverridden(prop, values, inheritedValues)}
        {@const inheritedVal = resolveInheritedVal(prop, inheritedValues)}

        <PropertyRow
          property={prop}
          {isOverridden}
          inheritedValue={inheritedVal}
          scope={scope || 'global'}
          {disabled}
          on:reset={() => handleResetProperty(prop.key)}
        >
          {#if controlType === 'switch'}
            <SwitchControl
              property={prop}
              value={currentVal}
              {disabled}
              hideHeader={true}
              on:change={(e) => updatePropertyValue(prop.key, e.detail)}
            />
          {:else if controlType === 'select'}
            <SelectControl
              property={prop}
              value={currentVal}
              {disabled}
              hideHeader={true}
              on:change={(e) => updatePropertyValue(prop.key, e.detail)}
            />
          {:else if controlType === 'path'}
            <PathPickerControl
              property={prop}
              value={currentVal}
              {disabled}
              hideHeader={true}
              on:change={(e) => updatePropertyValue(prop.key, e.detail)}
            />
          {:else if controlType === 'env'}
            <EnvEditorControl
              property={prop}
              value={currentVal}
              {disabled}
              hideHeader={true}
              on:change={(e) => updatePropertyValue(prop.key, e.detail)}
            />
          {:else if controlType === 'slider'}
            <SliderControl
              property={prop}
              value={currentVal}
              {disabled}
              on:change={(e) => updatePropertyValue(prop.key, e.detail)}
            />
          {:else}
            <TextControl
              property={prop}
              value={currentVal}
              {disabled}
              on:change={(e) => updatePropertyValue(prop.key, e.detail)}
            />
          {/if}
        </PropertyRow>
      {/each}
    </div>
  {/if}
</div>
