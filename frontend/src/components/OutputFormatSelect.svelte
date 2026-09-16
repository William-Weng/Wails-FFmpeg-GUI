<script lang="ts">
    type Container = "mp4" | "mkv" | "webm" | "ts";

    const containers: Container[] = ["mp4", "mkv", "webm", "ts"];

    let { value = $bindable<Container>("mp4"), disabled = false } = $props();
</script>

<select class="select-input" bind:value {disabled} aria-label="輸出格式">
    {#each containers as container}
        <option value={container}>.{container}</option>
    {/each}
</select>
