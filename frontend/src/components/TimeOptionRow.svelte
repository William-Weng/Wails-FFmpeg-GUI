<script lang="ts">
    type Props = {
        label: string;
        ariaLabel: string;
        placeholder: string;
        enabled?: boolean;
        time?: string;
        converting?: boolean;
    };

    let {
        label,
        ariaLabel,
        placeholder,
        enabled = $bindable(false),
        time = $bindable(""),
        converting = false,
    }: Props = $props();
</script>

<div class="option-row">
    <label class="check-label">
        <input type="checkbox" bind:checked={enabled} disabled={converting} />

        <span>{label}</span>
    </label>

    <input
        class="number-input time"
        bind:value={time}
        disabled={!enabled || converting}
        {placeholder}
        aria-label={ariaLabel}
    />
</div>
