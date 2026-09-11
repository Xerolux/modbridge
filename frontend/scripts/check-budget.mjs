import { readdirSync, readFileSync } from 'node:fs';
import { gzipSync } from 'node:zlib';
const assets=new URL('../dist/assets/',import.meta.url);
const totals={js:0,css:0};
for(const name of readdirSync(assets)) {
 const kind=name.endsWith('.js')?'js':name.endsWith('.css')?'css':null;
 if(kind)totals[kind]+=gzipSync(readFileSync(new URL(name,assets))).length;
}
const budgets={js:500*1024,css:40*1024};
for(const kind of Object.keys(budgets)) {
 console.log(`${kind}: ${(totals[kind]/1024).toFixed(1)} KiB gzip / ${budgets[kind]/1024} KiB budget (all routes)`);
 if(totals[kind]>budgets[kind])process.exitCode=1;
}
