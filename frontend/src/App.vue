<template>
  <div class="app">
    <header class="app-header">
      <h1>LoRa Sensor Management</h1>
      <nav>
        <router-link to="/">Sensors</router-link>
        <router-link to="/add">+ Add Sensor</router-link>
      </nav>
    </header>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<style>
.app-header {
  background: #1a1a2e;
  color: white;
  padding: 1rem 2rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.app-header h1 { margin: 0; font-size: 1.25rem; }
.app-header nav { display: flex; gap: 1.5rem; }
.app-header a { color: #a8dadc; text-decoration: none; font-weight: 500; }
.app-header a:hover, .app-header a.router-link-active { color: white; }
.app-main { max-width: 960px; margin: 2rem auto; padding: 0 1rem; }
</style>
