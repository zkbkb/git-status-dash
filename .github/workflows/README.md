# GitHub Actions Workflows

本项目包含以下GitHub Actions工作流：

## 1. Test Build (`test_build.yml`)
- **触发方式**: 
  - 自动：每次push到任意分支
  - 手动：通过GitHub Actions页面的"Run workflow"按钮
- **功能**: 运行所有测试和构建检查
  - Go代码测试和多平台构建
  - Node.js代码lint和测试
  - 安全漏洞扫描
- **状态**: [![Test Build](https://github.com/zkbkb/git-status-dash/actions/workflows/test_build.yml/badge.svg)](https://github.com/zkbkb/git-status-dash/actions/workflows/test_build.yml)

## 2. Test Release (`test_release.yml`)
- **触发方式**: 
  - 手动：通过GitHub Actions页面的"Run workflow"按钮
  - 可以选择跳过测试（用于调试）
- **功能**: 测试完整的发布流程
  - 验证Node.js包构建
  - 构建所有平台的Go二进制文件
  - 创建测试版本的发布包
- **用途**: 在正式发布前验证发布流程是否正常工作

## 3. Production Release (`production_release.yml`)
- **触发方式**: 
  - 自动：创建以`v`开头的标签时（如`v1.0.0`）
- **功能**: 执行正式发布
  - 发布到npm registry
  - 构建所有平台的二进制文件
  - 创建GitHub Release
  - 上传所有构建产物

## 如何使用

### 日常开发
- 只需正常push代码，`Test Build`会自动运行

### 测试发布流程
1. 进入GitHub仓库的Actions页面
2. 选择"Test Release"
3. 点击"Run workflow"
4. 可选择是否跳过测试
5. 点击绿色的"Run workflow"按钮

### 正式发布
```bash
# 创建并推送版本标签
git tag v1.0.0
git push origin v1.0.0
```
这将自动触发Production Release流程。