let audioContext: AudioContext | null = null

function getContext(): AudioContext | null {
  if (typeof window === 'undefined') return null

  if (!audioContext) {
    const AudioContextClass = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext
    if (!AudioContextClass) return null
    audioContext = new AudioContextClass()
  }

  if (audioContext.state === 'suspended') {
    void audioContext.resume()
  }
  return audioContext
}

function noiseBurst(context: AudioContext, durationMs: number): AudioBufferSourceNode {
  const length = Math.max(1, Math.floor(context.sampleRate * (durationMs / 1000)))
  const buffer = context.createBuffer(1, length, context.sampleRate)
  const data = buffer.getChannelData(0)
  for (let i = 0; i < length; i++) {
    data[i] = Math.random() * 2 - 1
  }

  const source = context.createBufferSource()
  source.buffer = buffer
  return source
}

interface Layer {
  filterType: BiquadFilterType
  filterFreq: number
  filterQ: number
  durationMs: number
  peakGain: number
}

function playLayer(context: AudioContext, layer: Layer) {
  const source = noiseBurst(context, layer.durationMs)
  const filter = context.createBiquadFilter()
  filter.type = layer.filterType
  filter.frequency.value = layer.filterFreq
  filter.Q.value = layer.filterQ

  const gain = context.createGain()
  const now = context.currentTime
  gain.gain.setValueAtTime(layer.peakGain, now)
  gain.gain.exponentialRampToValueAtTime(0.001, now + layer.durationMs / 1000)

  source.connect(filter)
  filter.connect(gain)
  gain.connect(context.destination)
  source.start(now)
}

function playPieceSound(variant: 'move' | 'capture') {
  const context = getContext()
  if (!context) return

  const isCapture = variant === 'capture'

  playLayer(context, {
    filterType: 'bandpass',
    filterFreq: isCapture ? 2200 : 1600,
    filterQ: 1.1,
    durationMs: 18,
    peakGain: isCapture ? 0.5 : 0.35,
  })
  playLayer(context, {
    filterType: 'lowpass',
    filterFreq: isCapture ? 320 : 220,
    filterQ: 0.7,
    durationMs: isCapture ? 100 : 75,
    peakGain: isCapture ? 0.4 : 0.3,
  })
}

export function playMoveSound() {
  playPieceSound('move')
}

export function playCaptureSound() {
  playPieceSound('capture')
}
