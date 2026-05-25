const https = require('https');
const fs = require('fs');
const path = require('path');

// 图标配置
const icons = [
  { name: 'home', regular: 'home-24-regular', filled: 'home-24-filled' },
  { name: 'vehicle', regular: 'vehicle-car-24-regular', filled: 'vehicle-car-24-filled' },
  { name: 'control', regular: 'phone-desktop-24-regular', filled: 'phone-desktop-24-filled' },
  { name: 'profile', regular: 'person-24-regular', filled: 'person-24-filled' }
];

const colors = {
  regular: '999999',  // 灰色
  filled: 'e82127'    // 红色
};

const outputDir = path.join(__dirname, '..', 'static', 'tabbar');

// 确保输出目录存在
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

// 下载函数 - 使用 iconify 的 PNG API
function downloadPNG(iconName, style, color, outputName) {
  return new Promise((resolve, reject) => {
    // 使用 iconify 的 PNG 转换 API
    const url = `https://api.iconify.design/fluent/${iconName}.svg?color=%23${color}&width=48&height=48`;
    
    console.log(`Downloading: ${iconName} (${style}) -> ${outputName}`);
    
    // 先下载 SVG
    https.get(url, (response) => {
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download ${iconName}: ${response.statusCode}`));
        return;
      }
      
      let svgData = '';
      response.on('data', chunk => svgData += chunk);
      response.on('end', () => {
        // 将 SVG 转换为 PNG 使用 sharp
        const sharp = require('sharp');
        const svgBuffer = Buffer.from(svgData);
        
        sharp(svgBuffer)
          .resize(48, 48)
          .png()
          .toFile(path.join(outputDir, outputName))
          .then(() => {
            console.log(`✓ Created: ${outputName}`);
            resolve(outputName);
          })
          .catch(err => {
            console.error(`✗ Failed to convert ${outputName}:`, err.message);
            reject(err);
          });
      });
    }).on('error', reject);
  });
}

// 检查 sharp 是否安装
try {
  require('sharp');
} catch (e) {
  console.log('Installing sharp package for image conversion...');
  const { execSync } = require('child_process');
  execSync('npm install sharp --save-dev', { cwd: path.join(__dirname, '..'), stdio: 'inherit' });
}

// 主函数
async function main() {
  console.log('Downloading Fluent Icons as PNG...');
  console.log('=================================');
  console.log('');
  
  for (const icon of icons) {
    try {
      // 下载 regular 版本（灰色）
      await downloadPNG(icon.regular, 'regular', colors.regular, `${icon.name}.png`);
      
      // 下载 filled 版本（红色）
      await downloadPNG(icon.filled, 'filled', colors.filled, `${icon.name}-active.png`);
      
      console.log(`✓ Completed ${icon.name} icons\n`);
    } catch (error) {
      console.error(`✗ Failed ${icon.name}:`, error.message);
    }
  }
  
  console.log('');
  console.log('All icons downloaded to:', outputDir);
  console.log('');
  console.log('Files created:');
  const files = fs.readdirSync(outputDir);
  files.filter(f => f.endsWith('.png')).forEach(f => console.log('  -', f));
}

main().catch(console.error);
