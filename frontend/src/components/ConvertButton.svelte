<script lang="ts">
    type Props = {
        videoDuration?: number
        converting?: boolean;
        cancelling?: boolean;
        hasInput?: boolean;
        onclick?: () => void;
    };

    let {
        videoDuration = 0,
        converting = false,
        cancelling = false,
        hasInput = false,
        onclick,
    }: Props = $props();
</script>

<button
    class="convert-button"
    class:converting
    class:cancelling
    {onclick}
    disabled={(!hasInput && !converting) || cancelling || (videoDuration < 1)}
>
    {cancelling ? "取消中..." : converting ? "取消" : "轉換"}
</button>
