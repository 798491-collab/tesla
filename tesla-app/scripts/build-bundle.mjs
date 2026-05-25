import { readFileSync, writeFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');

const threeModule = await import('three');
const { DRACOLoader } = await import('three/examples/jsm/loaders/DRACOLoader.js');

const dracoSource = `
// DRACOLoader - injected into THREE_BUNDLE
var DRACOLoader = ${DRACOLoader.toString()};
`;

const bundlePath = join(root, 'static/three-bundle.js');
let bundle = readFileSync(bundlePath, 'utf8');

const exportLine = 'exports.BloomEffect = BloomEffect;';
const dracoExport = `
exports.DRACOLoader = DRACOLoader;`;

bundle = bundle.replace(
  exportLine,
  dracoSource + '\n' + exportLine + dracoExport
);

writeFileSync(bundlePath, bundle);
console.log('DRACOLoader injected into three-bundle.js');
