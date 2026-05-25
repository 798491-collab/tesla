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
  'VolumeHigh': 'VolumeHigh',
  'Bulb': 'Bulb',
  'Alarm': 'Alarm',
  'Sync': 'Sync',
  'Time': 'Time',
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

fs.writeFileSync(path.resolve(__dirname, '../utils/iconPaths.js'), output);

console.log(`Generated ${Object.keys(iconData).length} icons`);
if (notFound.length > 0) {
  console.log('\nNot found:');
  notFound.forEach(n => console.log(`  ${n}`));
}
