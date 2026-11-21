# Elise

Elise 是运行在 macOS 菜单栏的本地 TOTP 验证器。打开 Elise 后，点击菜单栏的 **ELISE OTP**。

- 选择 **添加账户…**，粘贴服务提供的 `otpauth://totp` 链接或输入 Base32 密钥。使用 Base32 密钥时需填写账户名称，默认使用 SHA-1、6 位验证码和 30 秒刷新周期。
- 账户旁显示当前验证码。点击账户所在行即可复制当时有效的验证码。
- 选择 **移除账户**，再点选需要移除的账户。账户和密钥保存在当前用户的 `~/Library/Application Support/Elise/accounts.json`，文件仅对该用户开放读写。生成验证码无需联网。

## 构建与安装

需要 macOS 15 或更新版本、Go、Xcode Command Line Tools。构建脚本生成与当前机器架构一致的本地签名应用。

```sh
go test ./...
./macos/build.sh "$HOME/Applications/Elise.app"
open "$HOME/Applications/Elise.app"
```

直接运行 `./macos/build.sh` 会把应用构建到 `dist/Elise.app`。构建目标须为新路径。更新已有的安装时，先退出 Elise，再用新构建的应用替换旧应用。
