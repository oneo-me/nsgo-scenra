// Synthesizes the promo video's music, so it needs no recording or licence:
// a pad, plucked arpeggios, bass and soft drums at 100 BPM, a whoosh into
// each cut and a last chord under the outro. Returns a 16-bit stereo WAV.

const RATE = 44100;
const BEAT = 0.6;
// The first downbeat, so the drums enter on the cover's cut at 2.6s.
const START = 0.2;

const note = (name) => {
  const [, letter, octave] = name.match(/^([A-G]#?)(\d)$/);
  const index = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B'].indexOf(letter);
  return 440 * Math.pow(2, (index + (Number(octave) + 1) * 12 - 69) / 12);
};

// Two bars each: Fmaj7, G6, Em7, Am7.
const chords = [
  ['F3', 'A3', 'C4', 'E4'],
  ['G3', 'B3', 'D4', 'E4'],
  ['E3', 'G3', 'B3', 'D4'],
  ['A3', 'C4', 'E4', 'G4'],
].map((c) => c.map(note));
const roots = ['F2', 'G2', 'E2', 'A2'].map(note);
const last = ['C3', 'G3', 'B3', 'D4', 'E4'].map(note);

export function music({ duration, cuts, outro }) {
  const n = Math.ceil(duration * RATE);
  const left = new Float32Array(n);
  const right = new Float32Array(n);
  const plucks = new Float32Array(n);

  const add = (buffer, at, length, fn) => {
    const from = Math.max(0, Math.floor(at * RATE));
    const to = Math.min(n, Math.floor((at + length) * RATE));
    for (let i = from; i < to; i++) buffer[i] += fn(i / RATE - at);
  };
  const both = (at, length, fn, pan = 0) => {
    add(left, at, length, (t) => fn(t) * (1 - pan));
    add(right, at, length, (t) => fn(t) * (1 + pan));
  };
  const tone = (f, t) => Math.sin(2 * Math.PI * f * t) + 0.3 * Math.sin(4 * Math.PI * f * t) + 0.12 * Math.sin(6 * Math.PI * f * t);
  let seed = 1;
  const noise = () => ((seed = (seed * 16807) % 2147483647) / 2147483647) * 2 - 1;

  // Pads, swelling over each chord, the last one under the outro.
  const chordAt = (t) => Math.floor(Math.max(0, t - START) / (8 * BEAT)) % chords.length;
  for (let at = START; at < outro; at += 8 * BEAT) {
    const begin = at === START ? 0 : at;
    const chord = chords[chordAt(begin + 0.01)];
    const length = Math.min(8 * BEAT, outro - begin) + 1.2;
    chord.forEach((f, i) => {
      for (const detune of [-0.25, 0.25]) {
        both(begin, length, (t) => {
          const env = Math.min(1, t / 0.5) * Math.min(1, (length - t) / 1.2);
          return 0.022 * env * tone(f + detune, t);
        }, (i % 2 ? 0.3 : -0.3) * Math.sign(detune));
      }
    });
  }
  last.forEach((f, i) => {
    both(outro, duration - outro, (t) => 0.03 * Math.min(1, t / 0.08) * Math.exp(-t / 2.4) * tone(f, t) * (1 + 0.4 * Math.exp(-t * 8)), i % 2 ? 0.25 : -0.25);
  });

  // Plucked arpeggios in eighths, an octave up, into the delay.
  for (let at = START, step = 0; at < outro; at += BEAT / 2, step++) {
    const chord = chords[chordAt(at + 0.01)];
    const f = chord[[0, 2, 1, 3, 2, 1, 3, 2][step % 8]] * 2;
    const gain = step % 2 ? 0.05 : 0.07;
    add(plucks, at, 0.8, (t) => gain * Math.min(1, t / 0.003) * Math.exp(-t * 7) * (Math.sin(2 * Math.PI * f * t) + 0.25 * Math.sin(4 * Math.PI * f * t) * Math.exp(-t * 15)));
  }

  // Bass, kick and hats from the first cut until the outro.
  for (let at = START + 4 * BEAT, beat = 0; at < outro - 0.05; at += BEAT, beat++) {
    const f = roots[chordAt(at + 0.01)];
    if (beat % 2 === 0) both(at, BEAT * 2, (t) => 0.11 * Math.min(1, t / 0.01) * Math.exp(-t * 1.6) * Math.sin(2 * Math.PI * f * t));
    both(at, 0.35, (t) => 0.32 * Math.exp(-t * 14) * Math.sin(2 * Math.PI * (45 * t + (75 / 30) * (1 - Math.exp(-t * 30)))));
    const pan = beat % 2 ? 0.4 : -0.4;
    both(at + BEAT / 2, 0.08, (t) => 0.035 * Math.exp(-t * 60) * noise(), pan);
  }

  // A rising whoosh into each cut and a low hit on it.
  for (const cut of [...cuts, outro]) {
    let low = 0;
    let band = 0;
    both(cut - 0.7, 0.75, (t) => {
      const k = t / 0.7;
      const f = 2 * Math.sin((Math.PI * (300 + 5000 * k * k)) / RATE);
      const high = noise() - low - 0.6 * band;
      band += f * high;
      low += f * band;
      return 0.07 * Math.pow(Math.min(1, k), 2) * band * (t < 0.7 ? 1 : 1 - (t - 0.7) / 0.05);
    });
    both(cut, 1.2, (t) => 0.22 * Math.exp(-t * 3.5) * Math.sin(2 * Math.PI * (38 * t + 2 * (1 - Math.exp(-t * 20)))));
  }

  // A dotted-eighth ping-pong delay on the plucks.
  const delay = Math.floor(BEAT * 0.75 * RATE);
  const dl = new Float32Array(n);
  const dr = new Float32Array(n);
  for (let i = 0; i < n; i++) {
    const back = i - delay;
    dl[i] = plucks[i] + (back >= 0 ? dr[back] * 0.45 : 0);
    dr[i] = back >= 0 ? dl[back] * 0.45 : 0;
    left[i] += plucks[i] * 0.8 + dl[i] * 0.35;
    right[i] += plucks[i] * 0.8 + dr[i] * 0.45;
  }

  // Soft clipping, a fade out, and the peak at -1 dBFS.
  let peak = 0;
  for (let i = 0; i < n; i++) {
    const fade = Math.min(1, i / (RATE * 0.05), (n - i) / (RATE * 2));
    left[i] = Math.tanh(left[i] * 1.4) * fade;
    right[i] = Math.tanh(right[i] * 1.4) * fade;
    peak = Math.max(peak, Math.abs(left[i]), Math.abs(right[i]));
  }
  const scale = 0.89 / peak;

  const wav = Buffer.alloc(44 + n * 4);
  wav.write('RIFF', 0);
  wav.writeUInt32LE(36 + n * 4, 4);
  wav.write('WAVEfmt ', 8);
  wav.writeUInt32LE(16, 16);
  wav.writeUInt16LE(1, 20);
  wav.writeUInt16LE(2, 22);
  wav.writeUInt32LE(RATE, 24);
  wav.writeUInt32LE(RATE * 4, 28);
  wav.writeUInt16LE(4, 32);
  wav.writeUInt16LE(16, 34);
  wav.write('data', 36);
  wav.writeUInt32LE(n * 4, 40);
  for (let i = 0; i < n; i++) {
    wav.writeInt16LE(Math.round(left[i] * scale * 32767), 44 + i * 4);
    wav.writeInt16LE(Math.round(right[i] * scale * 32767), 46 + i * 4);
  }
  return wav;
}
