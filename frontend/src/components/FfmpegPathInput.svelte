<script lang="ts">
  let { value = $bindable(""), disabled = false } = $props();
</script>

<div class="form-row ffmpeg-row">
  <input
    class="text-input ffmpeg-input"
    bind:value
    {disabled}
    aria-label="FFmpeg 路徑"
  />
</div>
