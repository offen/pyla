<script setup>
const model = defineModel()
const { label, placeholder, rows = 4} = defineProps(['label', 'placeholder', 'rows'])
</script>

<template>
  <label class="block mb-0 text-sm/5 text-neutral-500">
    {{ label }}
    <textarea
      v-bind="$attrs"
      v-model="model"
      class="block w-full rounded-lg p-4 text-base/7 bg-neutral-50"
      :rows="rows"
      :placeholder="placeholder"
    ></textarea>
  </label>
</template>
