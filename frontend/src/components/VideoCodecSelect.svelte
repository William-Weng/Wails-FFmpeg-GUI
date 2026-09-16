<script lang="ts">
    type VideoCodec = "copy" | "h264" | "h265";

    const codecs: { value: VideoCodec; label: string; }[] = [
        { value: "copy", label: "Copy" },
        { value: "h264", label: "H.264" },
        { value: "h265", label: "H.265" },
    ];

    let { value = $bindable<VideoCodec>("copy"), disabled = false } = $props();
</script>

<select class="select-input" bind:value {disabled} aria-label="影片編碼">
    {#each codecs as codec (codec.value)}
        <option value={codec.value}>{codec.label}</option>
    {/each}
</select>
