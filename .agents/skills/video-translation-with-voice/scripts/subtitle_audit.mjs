#!/usr/bin/env node
import { readFileSync } from 'node:fs';

const args = parseArgs(process.argv.slice(2));
const punctuation = /[，。？！、；：·・･]/u;

function parseArgs(argv) {
  const out = { ban: [], banRegex: [] };
  for (let i = 0; i < argv.length; i++) {
    const key = argv[i];
    const value = argv[i + 1];
    if (!key.startsWith('--')) continue;
    i++;
    if (key === '--ban') out.ban.push(value);
    else if (key === '--ban-regex') out.banRegex.push(value);
    else out[key.slice(2)] = value;
  }
  return out;
}

function readMaybe(file) {
  return file ? readFileSync(file, 'utf8') : '';
}

function loadGlossary(file) {
  if (!file) return {};
  return JSON.parse(readFileSync(file, 'utf8'));
}

function displayWidth(text) {
  let width = 0;
  for (const ch of [...text]) width += /[^\x00-\xff]/u.test(ch) ? 1 : 0.55;
  return width;
}

function parseSrt(srt) {
  if (!srt.trim()) return [];
  return srt.trim().split(/\n\s*\n/u).map((block) => {
    const lines = block.split('\n');
    return {
      index: lines[0]?.trim(),
      time: lines[1]?.trim(),
      text: lines.slice(2).join('').trim(),
    };
  });
}

function checkText(label, text, issues) {
  if (punctuation.test(text)) {
    issues.push(`${label}: contains Chinese punctuation`);
  }
  for (const term of args.ban) {
    if (term && text.includes(term)) {
      issues.push(`${label}: contains banned term ${JSON.stringify(term)}`);
    }
  }
  for (const pattern of args.banRegex) {
    if (pattern && new RegExp(pattern, 'u').test(text)) {
      issues.push(`${label}: matches banned regex ${JSON.stringify(pattern)}`);
    }
  }
}

const srt = readMaybe(args.srt);
const ass = readMaybe(args.ass);
const glossary = loadGlossary(args.glossary);
const protectedNames = glossary.protected_names || [];
const issues = [];
const srtBlocks = parseSrt(srt);

let maxWidth = 0;
for (const cue of srtBlocks) {
  checkText(`SRT cue ${cue.index}`, cue.text, issues);
  const width = displayWidth(cue.text);
  maxWidth = Math.max(maxWidth, width);
  if (width > Number(args.maxWidth || 31)) {
    issues.push(`SRT cue ${cue.index}: display width ${width.toFixed(1)} exceeds ${args.maxWidth || 31}`);
  }
}

if (ass) {
  checkText('ASS', ass, issues);
  for (const name of protectedNames) {
    const escapedParts = name.split(/\s+/u).map((part) => part.replace(/[.*+?^${}()|[\]\\]/gu, '\\$&'));
    if (escapedParts.length < 2) continue;
    const brokenPattern = escapedParts.join(String.raw`\\N+`);
    if (new RegExp(brokenPattern, 'u').test(ass)) {
      issues.push(`ASS: protected name appears line-broken: ${name}`);
    }
  }
}

const result = {
  ok: issues.length === 0,
  cues: srtBlocks.length,
  maxWidth: Number(maxWidth.toFixed(1)),
  issues,
};

console.log(JSON.stringify(result, null, 2));
process.exit(result.ok ? 0 : 1);
