<script setup>
const model = defineModel()
const { label, placeholder, rows = 4} = defineProps(['label', 'placeholder', 'rows'])
</script>

<template>
  <label class="block mb-0 font-mono text-sm/5 text-neutral-500">
    {{ label }}
    <textarea
      v-bind="$attrs"
      v-model="model"
      class="h-40 block w-full rounded-lg p-3 mt-2 text-base/6 text-neutral-300 bg-neutral-700"
      :rows="rows"
      :placeholder="placeholder"
    ></textarea>
  </label>
</template>