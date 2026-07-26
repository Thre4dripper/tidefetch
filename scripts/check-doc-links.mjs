import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = path.resolve(import.meta.dirname, '..');
const markdownFiles = [];

function collect(directory) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    if (['.git', 'node_modules', 'dist'].includes(entry.name)) continue;
    const fullPath = path.join(directory, entry.name);
    if (entry.isDirectory()) collect(fullPath);
    else if (entry.name.endsWith('.md')) markdownFiles.push(fullPath);
  }
}

collect(root);

const broken = [];
const markdownLink = /!?\[[^\]]*\]\(([^)]+)\)/g;

for (const file of markdownFiles) {
  const source = fs.readFileSync(file, 'utf8');
  for (const match of source.matchAll(markdownLink)) {
    let target = match[1].trim().replace(/^<|>$/g, '');
    if (!target || /^(https?:|mailto:|#)/.test(target)) continue;
    target = target.split('#')[0].split('?')[0];
    const resolved = path.resolve(path.dirname(file), decodeURIComponent(target));
    if (!fs.existsSync(resolved)) {
      broken.push(`${path.relative(root, file)} -> ${target}`);
    }
  }
}

if (broken.length > 0) {
  console.error(`Broken relative Markdown links:\n${broken.join('\n')}`);
  process.exit(1);
}

console.log(`Validated ${markdownFiles.length} Markdown files; all relative links resolve.`);