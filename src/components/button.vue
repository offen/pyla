<script setup>
const { type } = defineProps(['type'])
</script>

<template>
  <button
    v-bind="$attrs"
    class="font-semibold outline-2 outline-neutral-950 px-4 py-2 rounded-lg w-64"
    :class="[
      $attrs.disabled ? 'pointer-events-none cursor-not-allowed' : 'cursor-pointer',
      type === 'fill' ? 'text-white bg-neutral-950 disabled:bg-neutral-400 disabled:outline-neutral-400' : '',
      type === 'outline' ? 'text-neutral-950 bg-transparent' : '',
    ]"
  >
    <slot />
  </button>
</template>
