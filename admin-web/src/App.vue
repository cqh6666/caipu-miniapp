<template>
  <router-view v-slot="{ Component, route: currentRoute }">
    <transition name="route-fade" mode="out-in">
      <component :is="Component" :key="currentRoute.path" />
    </transition>
  </router-view>
</template>
