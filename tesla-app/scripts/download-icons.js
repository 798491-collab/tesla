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

// 下载 SVG 并转换为 PNG 的简单实现
// 使用 sharp 库进行转换
async function downloadIcon(iconName, style, color) {
  const url = `https://api.iconify.design/fluent/${iconName}.svg?color=%23${color}`;
  const outputFile = path.join(outputDir, `${iconName.split('-')[0]}${style === 'filled' ? '-active' : ''}.svg`);
  
  return new Promise((resolve, reject) => {
    https.get(url, (response) => {
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download ${iconName}: ${response.statusCode}`));
        return;
      }
      
      let data = '';
      response.on('data', chunk => data += chunk);
      response.on('end', () => {
        fs.writeFileSync(outputFile, data);
        console.log(`Downloaded: ${outputFile}`);
        resolve(outputFile);
      });
    }).on('error', reject);
  });
}

// 主函数
async function main() {
  console.log('Starting icon download...');
  console.log(`Output directory: ${outputDir}`);
  
  for (const icon of icons) {
    try {
      // 下载 regular 版本（灰色）
      await downloadIcon(icon.regular, 'regular', colors.regular);
      
      // 下载 filled 版本（红色）
      await downloadIcon(icon.filled, 'filled', colors.filled);
      
      console.log(`✓ Downloaded ${icon.name} icons`);
    } catch (error) {
      console.error(`✗ Failed to download ${icon.name}:`, error.message);
    }
  }
  
  console.log('\nDownload complete!');
  console.log('Note: Downloaded SVG files. You may need to convert them to PNG for tabBar usage.');
}

main().catch(console.error);
