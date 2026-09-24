<script setup lang="ts">
const props = defineProps<{ title: string; snippet: string; query: string }>();

function highlight(text: string, q: string) {
  if (!q) return text;
  return text.split(q).join(`<mark>${q}</mark>`);
}
</script>

<template>
  <article class="result">
    <h3>{{ props.title }}</h3>
    <p v-html="highlight(props.snippet, props.query)"></p>
  </article>
</template>
