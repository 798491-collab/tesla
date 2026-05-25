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
const size = 48; // 图标尺寸

// 确保输出目录存在
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

// 创建 HTML 文件用于在浏览器中转换 SVG 为 PNG
function createConverterHTML() {
  const html = `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>SVG to PNG Converter</title>
  <style>
    body { font-family: Arial, sans-serif; padding: 20px; }
    .icon-item { margin: 10px 0; padding: 10px; border: 1px solid #ddd; }
    canvas { border: 1px solid #eee; margin: 5px; }
    .download-btn { 
      padding: 8px 16px; 
      background: #e82127; 
      color: white; 
      border: none; 
      cursor: pointer;
      margin: 5px;
    }
    .download-btn:hover { background: #c41a1f; }
  </style>
</head>
<body>
  <h1>Fluent Icons - SVG to PNG Converter</h1>
  <p>点击按钮下载 PNG 图标（${size}x${size} 像素）</p>
  <div id="icons"></div>
  
  <script>
    const icons = ${JSON.stringify(icons)};
    const colors = ${JSON.stringify(colors)};
    const size = ${size};
    
    async function loadSVG(url) {
      const response = await fetch(url);
      return await response.text();
    }
    
    function svgToPNG(svgContent, color, filename) {
      return new Promise((resolve) => {
        const canvas = document.createElement('canvas');
        canvas.width = size;
        canvas.height = size;
        const ctx = canvas.getContext('2d');
        
        const img = new Image();
        const svgBlob = new Blob([svgContent], { type: 'image/svg+xml' });
        const url = URL.createObjectURL(svgBlob);
        
        img.onload = () => {
          ctx.drawImage(img, 0, 0, size, size);
          URL.revokeObjectURL(url);
          
          canvas.toBlob((blob) => {
            resolve({ blob, canvas, filename });
          }, 'image/png');
        };
        
        img.src = url;
      });
    }
    
    function downloadPNG(blob, filename) {
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    }
    
    async function renderIcons() {
      const container = document.getElementById('icons');
      
      for (const icon of icons) {
        const div = document.createElement('div');
        div.className = 'icon-item';
        div.innerHTML = '<h3>' + icon.name + '</h3>';
        
        // Regular (灰色)
        const regularUrl = 'https://api.iconify.design/fluent/' + icon.regular + '.svg?color=%23' + colors.regular;
        const regularSVG = await loadSVG(regularUrl);
        const regularResult = await svgToPNG(regularSVG, colors.regular, icon.name + '.png');
        
        const regularCanvas = regularResult.canvas;
        const regularBtn = document.createElement('button');
        regularBtn.className = 'download-btn';
        regularBtn.textContent = '下载 ' + icon.name + '.png (灰色)';
        regularBtn.onclick = () => downloadPNG(regularResult.blob, icon.name + '.png');
        
        // Filled (红色)
        const filledUrl = 'https://api.iconify.design/fluent/' + icon.filled + '.svg?color=%23' + colors.filled;
        const filledSVG = await loadSVG(filledUrl);
        const filledResult = await svgToPNG(filledSVG, colors.filled, icon.name + '-active.png');
        
        const filledCanvas = filledResult.canvas;
        const filledBtn = document.createElement('button');
        filledBtn.className = 'download-btn';
        filledBtn.textContent = '下载 ' + icon.name + '-active.png (红色)';
        filledBtn.onclick = () => downloadPNG(filledResult.blob, icon.name + '-active.png');
        
        div.appendChild(regularCanvas);
        div.appendChild(regularBtn);
        div.appendChild(filledCanvas);
        div.appendChild(filledBtn);
        container.appendChild(div);
      }
    }
    
    renderIcons();
  <\/script>
</body>
</html>`;
  
  const htmlPath = path.join(__dirname, 'icon-converter.html');
  fs.writeFileSync(htmlPath, html);
  console.log('Created converter HTML: ' + htmlPath);
  console.log('Please open this file in your browser to download PNG icons.');
}

// 创建简单的 SVG 下载脚本（备用方案）
function createSVGDownloadScript() {
  const scriptContent = [
    '#!/bin/bash',
    '# SVG Download Script',
    '# Run this in Git Bash or WSL',
    '',
    'OUTPUT_DIR="../static/tabbar"',
    'mkdir -p $OUTPUT_DIR',
    '',
    '# Colors',
    'REGULAR_COLOR="999999"',
    'FILLED_COLOR="e82127"',
    '',
    'echo "Downloading Fluent Icons..."',
    '',
    '# home icons',
    'curl -L "https://api.iconify.design/fluent/home-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/home.svg"',
    'curl -L "https://api.iconify.design/fluent/home-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/home-active.svg"',
    '',
    '# vehicle icons',
    'curl -L "https://api.iconify.design/fluent/vehicle-car-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/vehicle.svg"',
    'curl -L "https://api.iconify.design/fluent/vehicle-car-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/vehicle-active.svg"',
    '',
    '# control icons',
    'curl -L "https://api.iconify.design/fluent/phone-desktop-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/control.svg"',
    'curl -L "https://api.iconify.design/fluent/phone-desktop-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/control-active.svg"',
    '',
    '# profile icons',
    'curl -L "https://api.iconify.design/fluent/person-24-regular.svg?color=%23${REGULAR_COLOR}" -o "$OUTPUT_DIR/profile.svg"',
    'curl -L "https://api.iconify.design/fluent/person-24-filled.svg?color=%23${FILLED_COLOR}" -o "$OUTPUT_DIR/profile-active.svg"',
    '',
    'echo "Done! SVG files saved to $OUTPUT_DIR"',
    'echo "Note: You need to convert SVG to PNG for tabBar usage."'
  ].join('\n');
  
  const scriptPath = path.join(__dirname, 'download-svg.sh');
  fs.writeFileSync(scriptPath, scriptContent);
  console.log('Created bash script: ' + scriptPath);
}

// 主函数
console.log('Fluent Icons Download Tool');
console.log('==========================');
console.log('');

// 方案1：创建 HTML 转换器
createConverterHTML();
console.log('');

// 方案2：创建 Bash 脚本（适用于 Git Bash/WSL）
createSVGDownloadScript();
console.log('');

console.log('使用方法：');
console.log('1. 打开 icon-converter.html 文件，在浏览器中预览并下载 PNG 图标');
console.log('2. 或者使用 Git Bash 运行 download-svg.sh 下载 SVG 文件');
console.log('');
console.log('图标将保存到: ' + outputDir);
