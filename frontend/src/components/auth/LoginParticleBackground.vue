<template>
  <div class="login-particle-bg" aria-hidden="true">
    <div class="login-particle-bg__glow login-particle-bg__glow--a" />
    <div class="login-particle-bg__glow login-particle-bg__glow--b" />
    <div class="login-particle-bg__glow login-particle-bg__glow--c" />
    <canvas ref="canvasRef" class="login-particle-bg__canvas" />
    <div class="login-particle-bg__noise" />
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

interface Particle {
  x: number
  y: number
  vx: number
  vy: number
  size: number
  alpha: number
}

const canvasRef = ref<HTMLCanvasElement | null>(null)
let rafId = 0
let particles: Particle[] = []
let width = 0
let height = 0
let dpr = 1
let mouseX = 0
let mouseY = 0

function prefersReducedMotion() {
  return typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function initParticles(count: number) {
  particles = Array.from({ length: count }, () => ({
    x: Math.random() * width,
    y: Math.random() * height,
    vx: (Math.random() - 0.5) * 0.35,
    vy: (Math.random() - 0.5) * 0.35,
    size: Math.random() * 1.6 + 0.6,
    alpha: Math.random() * 0.45 + 0.15,
  }))
}

function resize() {
  const canvas = canvasRef.value
  if (!canvas) return
  dpr = Math.min(window.devicePixelRatio || 1, 2)
  width = canvas.clientWidth
  height = canvas.clientHeight
  canvas.width = Math.floor(width * dpr)
  canvas.height = Math.floor(height * dpr)
  const ctx = canvas.getContext('2d')
  if (ctx) {
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  }
  if (particles.length === 0) {
    initParticles(width > 900 ? 90 : 60)
  }
}

function draw() {
  const canvas = canvasRef.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  if (!ctx) return

  ctx.clearRect(0, 0, width, height)

  for (const p of particles) {
    p.x += p.vx
    p.y += p.vy
    if (p.x < 0 || p.x > width) p.vx *= -1
    if (p.y < 0 || p.y > height) p.vy *= -1

    const dx = mouseX - p.x
    const dy = mouseY - p.y
    const dist = Math.hypot(dx, dy)
    if (dist < 120) {
      p.x -= dx * 0.008
      p.y -= dy * 0.008
    }

    ctx.beginPath()
    ctx.fillStyle = `rgba(147, 197, 253, ${p.alpha})`
    ctx.arc(p.x, p.y, p.size, 0, Math.PI * 2)
    ctx.fill()
  }

  for (let i = 0; i < particles.length; i += 1) {
    for (let j = i + 1; j < particles.length; j += 1) {
      const a = particles[i]
      const b = particles[j]
      const dist = Math.hypot(a.x - b.x, a.y - b.y)
      if (dist > 110) continue
      ctx.beginPath()
      ctx.strokeStyle = `rgba(96, 165, 250, ${0.14 * (1 - dist / 110)})`
      ctx.lineWidth = 1
      ctx.moveTo(a.x, a.y)
      ctx.lineTo(b.x, b.y)
      ctx.stroke()
    }
  }
}

function tick() {
  draw()
  rafId = window.requestAnimationFrame(tick)
}

function onMouseMove(event: MouseEvent) {
  mouseX = event.clientX
  mouseY = event.clientY
}

onMounted(() => {
  resize()
  window.addEventListener('resize', resize)
  window.addEventListener('mousemove', onMouseMove, { passive: true })
  if (!prefersReducedMotion()) {
    tick()
  } else {
    draw()
  }
})

onBeforeUnmount(() => {
  window.cancelAnimationFrame(rafId)
  window.removeEventListener('resize', resize)
  window.removeEventListener('mousemove', onMouseMove)
})
</script>

<style scoped lang="less">
.login-particle-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
  background: #050505;
  z-index: 0;
}

.login-particle-bg__canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.login-particle-bg__glow {
  position: absolute;
  filter: blur(80px);
  opacity: 0.55;
  border-radius: 50%;
  pointer-events: none;
  will-change: transform;
}

.login-particle-bg__glow--a {
  width: 55vw;
  height: 28vh;
  top: 8%;
  left: -8%;
  background: linear-gradient(120deg, rgba(255, 90, 60, 0.85), rgba(255, 160, 90, 0.35));
  animation: drift-a 18s ease-in-out infinite alternate;
}

.login-particle-bg__glow--b {
  width: 48vw;
  height: 32vh;
  top: 42%;
  right: -12%;
  background: linear-gradient(200deg, rgba(239, 68, 68, 0.65), rgba(251, 146, 60, 0.2));
  animation: drift-b 22s ease-in-out infinite alternate;
}

.login-particle-bg__glow--c {
  width: 40vw;
  height: 24vh;
  bottom: -6%;
  left: 28%;
  background: linear-gradient(160deg, rgba(59, 130, 246, 0.35), rgba(16, 185, 129, 0.15));
  animation: drift-c 26s ease-in-out infinite alternate;
}

.login-particle-bg__noise {
  position: absolute;
  inset: 0;
  opacity: 0.04;
  pointer-events: none;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}

@keyframes drift-a {
  from { transform: translate(0, 0) rotate(-8deg); }
  to { transform: translate(6vw, 4vh) rotate(4deg); }
}

@keyframes drift-b {
  from { transform: translate(0, 0) rotate(6deg); }
  to { transform: translate(-5vw, -3vh) rotate(-5deg); }
}

@keyframes drift-c {
  from { transform: translate(0, 0) rotate(0deg); }
  to { transform: translate(3vw, -2vh) rotate(8deg); }
}

@media (prefers-reduced-motion: reduce) {
  .login-particle-bg__glow {
    animation: none;
  }
}
</style>
