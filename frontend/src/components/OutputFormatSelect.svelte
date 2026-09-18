<script lang="ts">
    import { Containers } from "../constants/media";
    import type { Container } from "../constants/media";

    const containers: readonly Container[] = [
        Containers.MP4,
        Containers.MKV,
        Containers.TS,
        Containers.MP3,
        Containers.AAC,
        Containers.OGG,
    ];

    let { value = $bindable<Container>(containers[0]), disabled = false } = $props();
</script>

<select class="select-input" bind:value {disabled} aria-label="輸出格式">
    {#each containers as container}
        <option value={container}>.{container}</option>
    {/each}
</select>
