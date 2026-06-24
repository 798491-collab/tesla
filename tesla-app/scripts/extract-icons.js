const fs = require('fs');
const path = require('path');

const ioniconsDir = path.resolve(__dirname, '../node_modules/@vicons/ionicons5/es');

const iconMapping = {
  'Car': 'CarSport',
  'CarOutline': 'CarOutline',
  'CarSport': 'CarSport',
  'BatteryFull': 'BatteryFull',
  'BatteryHalf': 'BatteryHalf',
  'BatteryCharging': 'BatteryCharging',
  'Battery': 'BatteryHalf',
  'Flash': 'Flash',
  'FlashOutline': 'FlashOutline',
  'FlashOff': 'FlashOff',
  'LockClosed': 'LockClosed',
  'LockOpen': 'LockOpen',
  'Shield': 'Shield',
  'ShieldOutline': 'ShieldOutline',
  'Snow': 'Snow',
  'Sunny': 'Sunny',
  'SunnyOutline': 'SunnyOutline',
  'Thermometer': 'Thermometer',
  'ThermometerOutline': 'ThermometerOutline',
  'Location': 'Location',
  'LocationOutline': 'LocationOutline',
  'Navigate': 'Navigate',
  'NavigateOutline': 'NavigateOutline',
  'Speedometer': 'Speedometer',
  'SpeedometerOutline': 'SpeedometerOutline',
  'Person': 'Person',
  'PersonOutline': 'PersonOutline',
  'PersonAdd': 'PersonAdd',
  'Add': 'Add',
  'AddOutline': 'AddOutline',
  'ChevronDown': 'ChevronDown',
  'ChevronDownOutline': 'ChevronDownOutline',
  'ChevronForward': 'ChevronForward',
  'ChevronForwardOutline': 'ChevronForwardOutline',
  'ChevronBack': 'ChevronBack',
  'ChevronBackOutline': 'ChevronBackOutline',
  'ChevronUp': 'ChevronUp',
  'ChevronUpOutline': 'ChevronUpOutline',
  'ChevronRight': 'ChevronForward',
  'ChevronRightOutline': 'ChevronForwardOutline',
  'VolumeHigh': 'VolumeHigh',
  'Bulb': 'Bulb',
  'Alarm': 'Alarm',
  'Sync': 'Sync',
  'Time': 'Time',
  'TimeOutline': 'TimeOutline',
  'Calendar': 'Calendar',
  'Power': 'Power',
  'Walk': 'Walk',
  'Home': 'Home',
  'HomeOutline': 'HomeOutline',
  'InformationCircle': 'InformationCircle',
  'LogOut': 'LogOut',
  'Scan': 'Scan',
  'Key': 'Key',
  'Settings': 'Settings',
  'Trash': 'Trash',
  'HelpCircle': 'HelpCircle',
  'Ellipse': 'Ellipse',
  'EllipseOutline': 'EllipseOutline',
  'Exit': 'Exit',
  'ExitOutline': 'ExitOutline',
  'Desktop': 'Desktop',
  'DesktopOutline': 'DesktopOutline',
  'Bluetooth': 'Bluetooth',
  'Cloud': 'Cloud',
  'Warning': 'Warning',
  'Flame': 'Flame',
  'Map': 'Map',
  'Eye': 'Eye',
  'EyeOff': 'EyeOff',
  'Radio': 'Radio',
  'MusicNote': 'MusicalNote',
  'MusicalNotes': 'MusicalNotes',
  'Play': 'Play',
  'Pause': 'Pause',
  'PlayForward': 'PlayForward',
  'PlaySkipForward': 'PlaySkipForward',
  'PlaySkipBack': 'PlaySkipBack',
  'Wallet': 'Wallet',
  'WalletOutline': 'WalletOutline',
  'Cash': 'Cash',
  'CashOutline': 'CashOutline',
  'Sparkles': 'Sparkles',
  'CheckmarkCircle': 'CheckmarkCircle',
  'CheckmarkCircleOutline': 'CheckmarkCircleOutline',
  'ColorPalette': 'ColorPalette',
  'Create': 'Create',
  'Open': 'Open',
  'Globe': 'Globe',
  'Moon': 'Moon',
  'MoonOutline': 'MoonOutline',
  'Pricetag': 'Pricetag',
  'PricetagOutline': 'PricetagOutline',
  'StatsChart': 'StatsChart',
  'StatsChartOutline': 'StatsChartOutline',
  'TrendingUp': 'TrendingUp',
  'TrendingDown': 'TrendingDown',
  'Fan': 'Fan',
};

function extractSvgContent(filePath) {
  const content = fs.readFileSync(filePath, 'utf-8');

  const viewBoxMatch = content.match(/viewBox:\s*'([^']+)'/);
  const viewBox = viewBoxMatch ? viewBoxMatch[1] : '0 0 512 512';

  const staticMatch = content.match(/_createStaticVNode\('([\s\S]+?)',\s*\d+\)/);
  if (staticMatch) {
    let html = staticMatch[1];
    html = html.replace(/<\/(?:path|rect|circle|line|polyline|ellipse|polygon|g)>/g, '');
    html = html.replace(/<(path|rect|circle|line|polyline|ellipse|polygon|g)\b([^>]*?)>/g, '<$1$2/>');
    return { viewBox, content: html };
  }

  const elements = [];
  const nodeRegex = /_createElementVNode\(\s*'(\w+)'\s*,\s*\{([\s\S]*?)\}\s*,\s*null\s*,\s*-1/g;
  let match;
  while ((match = nodeRegex.exec(content)) !== null) {
    const tag = match[1];
    const attrsStr = match[2];

    const attrs = [];
    const attrRegex = /(?:(\w+)|'([^']+)')\s*:\s*'([^']*)'/g;
    let attrMatch;
    while ((attrMatch = attrRegex.exec(attrsStr)) !== null) {
      const key = attrMatch[1] || attrMatch[2];
      const value = attrMatch[3];
      attrs.push(`${key}="${value}"`);
    }

    elements.push(`<${tag} ${attrs.join(' ')}/>`);
  }

  return { viewBox, content: elements.join('') };
}

const iconData = {};
const notFound = [];

for (const [alias, fileName] of Object.entries(iconMapping)) {
  const filePath = path.join(ioniconsDir, `${fileName}.js`);
  if (!fs.existsSync(filePath)) {
    notFound.push(`${alias} -> ${fileName}`);
    continue;
  }

  try {
    const data = extractSvgContent(filePath);
    if (data.content) {
      iconData[alias] = {
        viewBox: data.viewBox,
        content: data.content,
      };
    } else {
      notFound.push(`${alias} -> ${fileName} (no content found)`);
    }
  } catch (e) {
    notFound.push(`${alias} -> ${fileName} (error: ${e.message})`);
  }
}

const output = `// Auto-generated from @vicons/ionicons5 by scripts/extract-icons.js
// Do not edit manually - run 'node scripts/extract-icons.js' to regenerate

export default ${JSON.stringify(iconData, null, 2)}
`;

// 手动添加的图标（ionicons5 中不存在，重新生成时需保留）
const manualIcons = {
  "Window": {
    "viewBox": "0 0 512 512",
    "content": "<rect x=\"96\" y=\"64\" width=\"320\" height=\"280\" rx=\"32\" ry=\"32\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"32\"/><line x1=\"96\" y1=\"200\" x2=\"416\" y2=\"200\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"32\"/><line x1=\"256\" y1=\"64\" x2=\"256\" y2=\"200\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"32\"/><path d=\"M176 448l32-72\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"24\"/><path d=\"M336 448l-32-72\" fill=\"none\" stroke=\"currentColor\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-width=\"24\"/>"
  },
  "Fan": {
    "viewBox": "0 0 512 512",
    "content": "<path d=\"M352 246c0-29.49-11.83-56.19-31-75.68v-50.45a6 6 0 0 0-9.66-4.74l-90.32 75.42A102.09 102.09 0 0 0 192 256c0 56.75 46.25 103 103 103c25.93 0 49.55-9.66 67.64-25.55l11.95-10.26a6 6 0 0 0 2.22-4.64v-13.18c-28.12-9.33-48.8-35.38-48.8-66.37z\" fill=\"currentColor\"/><path d=\"M462.35 193.27a102.14 102.14 0 0 0-58.88-56.14l-10.26-3.83a6 6 0 0 0-7.85 3.33l-45.12 90.25c-11.69 23.37-10.16 51.28 3.88 73.25a6 6 0 0 0 7.75 2.37l13.18-4.39c21.97-14.04 35.03-38.21 35.03-64.84z\" fill=\"currentColor\"/><path d=\"M142.2 269.35l-3.83-10.26a6 6 0 0 0-3.33-7.85l-90.25-45.12c-23.37-11.69-51.28-10.16-73.25 3.88a6 6 0 0 0-2.37 7.75l4.39 13.18c14.04 21.97 38.21 35.03 64.84 35.03c29.49 0 56.19-11.83 75.68-31v-50.45a6 6 0 0 0-4.74-9.66z\" fill=\"currentColor\"/><path d=\"M256 144c56.75 0 103-46.25 103-103c0-25.93-9.66-49.55-25.55-67.64l-10.26-11.95a6 6 0 0 0-4.64-2.22h-13.18c-9.33 28.12-35.38 48.8-66.37 48.8c-29.49 0-56.19-11.83-75.68-31l-75.42 90.32a6 6 0 0 0-4.74 9.66v50.45a102.09 102.09 0 0 0 29.71 20.98l90.32-75.42A102.14 102.14 0 0 0 256 144z\" fill=\"currentColor\"/><circle cx=\"256\" cy=\"288\" r=\"16\" fill=\"currentColor\"/>"
  },
  "Plug": {
    "viewBox": "0 0 512 512",
    "content": "<path d=\"M179.2 230.4c0-14.14 11.46-25.6 25.6-25.6h102.4c14.14 0 25.6 11.46 25.6 25.6v51.2c0 42.35-34.25 76.8-76.8 76.8s-76.8-34.45-76.8-76.8v-51.2z\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\" stroke-linejoin=\"round\"/><path d=\"M256 358.4V448\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/><path d=\"M204.8 204.8V128\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/><path d=\"M307.2 204.8V128\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/><path d=\"M160 448h192\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/>"
  },
  "Seat": {
    "viewBox": "0 0 512 512",
    "content": "<path d=\"M176 384V192c0-26.51 21.49-48 48-48h0c26.51 0 48 21.49 48 48v192\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\" stroke-linejoin=\"round\"/><path d=\"M144 384h224\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/><path d=\"M176 288h96\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/>"
  },
  "Steering": {
    "viewBox": "0 0 512 512",
    "content": "<circle cx=\"256\" cy=\"256\" r=\"160\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\"/><circle cx=\"256\" cy=\"256\" r=\"48\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\"/><path d=\"M256 304v112\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/><path d=\"M214 232L117 176\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/><path d=\"M298 232L395 176\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"32\" stroke-linecap=\"round\"/>"
  }
};

// 合并手动图标到自动生成的数据中
const finalData = { ...iconData, ...manualIcons };
const finalOutput = `// Auto-generated from @vicons/ionicons5 by scripts/extract-icons.js
// Do not edit manually - run 'node scripts/extract-icons.js' to regenerate

export default ${JSON.stringify(finalData, null, 2)}
`;

fs.writeFileSync(path.resolve(__dirname, '../utils/iconPaths.js'), finalOutput);

console.log(`Generated ${Object.keys(finalData).length} icons (${Object.keys(iconData).length} auto + ${Object.keys(manualIcons).length} manual)`);
if (notFound.length > 0) {
  console.log('\nNot found:');
  notFound.forEach(n => console.log(`  ${n}`));
}
