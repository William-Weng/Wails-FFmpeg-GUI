<script lang="ts">
  let { value = "", placeholder = "檔案路徑會顯示在這裡" } = $props();
</script>

<input
  class="text-input file-input"
  {value}
  readonly
  {placeholder}
  aria-label="輸入影片路徑"
/>
