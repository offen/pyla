<script setup>
const model = defineModel()
const { label, placeholder, rows = 4} = defineProps(['label', 'placeholder', 'rows'])
</script>

<template>
  <label class="block mb-0">
    {{ label }}
    <textarea
      v-bind="$attrs"
      v-model="model"
      class="block w-full bg-neutral-700 text-neutral-50"
      :rows="rows"
      :placeholder="placeholder"
    ></textarea>
  </label>
</template>