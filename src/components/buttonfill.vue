<script setup>
import { ref, useAttrs, computed } from 'vue'

const clicked = ref(false)
const $attrs = useAttrs()
const isDisabled = computed(() => !!$attrs.disabled)

const handleClick = (event) => {
  if (isDisabled.value) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  clicked.value = true
}
</script>

<template>
  <button
    v-bind="$attrs"
    :disabled="isDisabled"
    @click="handleClick"
    :class="[
      'text-white',
      'bg-neutral-950',
      'outline-neutral-950',
      'disabled:bg-neutral-400',
      'disabled:outline-neutral-400',
      'outline-2',
      'font-semibold',
      'w-64',
      'px-4',
      'py-2',
      'rounded-lg',
      isDisabled ? 'pointer-events-none cursor-not-allowed' : 'cursor-pointer'
    ]"
  >
    <slot />
  </button>
</template>