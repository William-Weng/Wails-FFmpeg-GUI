<script lang="ts">
    type VideoCodec = "copy" | "h264" | "h265";

    type Props = {
        enabled?: boolean;
        width?: number | null;
        height?: number | null;
        videoCodec?: VideoCodec;
        converting?: boolean;
    };

    let {
        enabled = $bindable(false),
        width = $bindable(null),
        height = $bindable(null),
        videoCodec = "copy",
        converting = false,
    }: Props = $props();

    let disabled = $derived(videoCodec === "copy" || converting);
    let inputDisabled = $derived(!enabled || disabled);
</script>

<div class="option-row">
    <label class="check-label">
        <input type="checkbox" bind:checked={enabled} {disabled} />

        <span>尺寸</span>
    </label>

    <input
        class="number-input"
        type="number"
        min="1"
        bind:value={width}
        disabled={inputDisabled}
        aria-label="輸出寬度"
    />

    <span class="separator">×</span>

    <input
        class="number-input"
        type="number"
        min="1"
        bind:value={height}
        disabled={inputDisabled}
        aria-label="輸出高度"
    />
</div>
